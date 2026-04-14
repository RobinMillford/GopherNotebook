package ingest

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/data/replication"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/yamin/gophernotebook/internal/db"
)

// IngestProgress reports the current state of ingestion for SSE streaming.
type IngestProgress struct {
	TotalFiles     int    `json:"totalFiles"`
	ProcessedFiles int    `json:"processedFiles"`
	CurrentFile    string `json:"currentFile"`
	Status         string `json:"status"` // "processing", "done", "error"
	Error          string `json:"error,omitempty"`
}

// FileJob represents a single file to be ingested.
type FileJob struct {
	FilePath   string
	NotebookID string
}

// WorkerPool manages concurrent document ingestion.
type WorkerPool struct {
	weaviateClient         *weaviate.Client
	embedder               *Embedder
	numWorkers             int
	semanticDedupThreshold float32
}

// NewWorkerPool creates a worker pool with NumCPU-1 workers (minimum 1).
// semanticDedupThreshold is a cosine distance threshold; set to 0 to disable dedup.
func NewWorkerPool(weaviateClient *weaviate.Client, embedder *Embedder, semanticDedupThreshold float32) *WorkerPool {
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	log.Printf("Worker pool initialized with %d workers (dedup threshold=%.3f)", workers, semanticDedupThreshold)

	return &WorkerPool{
		weaviateClient:         weaviateClient,
		embedder:               embedder,
		numWorkers:             workers,
		semanticDedupThreshold: semanticDedupThreshold,
	}
}

// Ingest processes a batch of files concurrently and reports progress via callback.
// The progressFn is called for every file completion (success or failure).
func (wp *WorkerPool) Ingest(ctx context.Context, jobs []FileJob, progressFn func(IngestProgress)) {
	total := len(jobs)
	var processed int64

	jobCh := make(chan FileJob, len(jobs))
	var wg sync.WaitGroup

	// Push all jobs to the channel
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	// Start workers
	for i := 0; i < wp.numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for job := range jobCh {
				select {
				case <-ctx.Done():
					return
				default:
				}

				log.Printf("[Worker %d] Processing: %s", workerID, job.FilePath)

				err := wp.processFile(ctx, job)

				current := atomic.AddInt64(&processed, 1)

				progress := IngestProgress{
					TotalFiles:     total,
					ProcessedFiles: int(current),
					CurrentFile:    job.FilePath,
					Status:         "processing",
				}

				if err != nil {
					log.Printf("[Worker %d] Error processing %s: %v", workerID, job.FilePath, err)
					progress.Error = err.Error()
				}

				if int(current) >= total {
					progress.Status = "done"
				}

				if progressFn != nil {
					progressFn(progress)
				}
			}
		}(i)
	}

	wg.Wait()
}

// processFile parses a file then delegates to processDoc.
func (wp *WorkerPool) processFile(ctx context.Context, job FileJob) error {
	doc, err := ParseFile(job.FilePath)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}
	return wp.processDoc(ctx, job.NotebookID, doc)
}

// IngestDoc ingests a pre-parsed document (e.g. from URL fetch) directly.
func (wp *WorkerPool) IngestDoc(ctx context.Context, notebookID string, doc *ParsedDocument) error {
	return wp.processDoc(ctx, notebookID, doc)
}

// processDoc runs Chunk → Embed → (optional dedup) → Batch Insert for a parsed document.
func (wp *WorkerPool) processDoc(ctx context.Context, notebookID string, doc *ParsedDocument) error {
	chunks := ChunkDocument(doc)
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks produced from %s", doc.FileName)
	}

	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}

	vectors, err := wp.embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	if len(vectors) != len(chunks) {
		return fmt.Errorf("vector count mismatch: got %d vectors for %d chunks", len(vectors), len(chunks))
	}

	// Semantic deduplication: skip chunks too similar to existing ones.
	if wp.semanticDedupThreshold > 0 {
		keep := make([]bool, len(chunks))
		sem := make(chan struct{}, wp.numWorkers)
		var wg sync.WaitGroup
		for i := range chunks {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				isDup, dedupErr := db.IsNearDuplicate(ctx, wp.weaviateClient, notebookID, vectors[i], wp.semanticDedupThreshold)
				keep[i] = dedupErr != nil || !isDup
			}(i)
		}
		wg.Wait()

		var filteredChunks []Chunk
		var filteredVectors [][]float32
		skipped := 0
		for i, k := range keep {
			if k {
				filteredChunks = append(filteredChunks, chunks[i])
				filteredVectors = append(filteredVectors, vectors[i])
			} else {
				skipped++
			}
		}
		if skipped > 0 {
			log.Printf("Semantic dedup: skipped %d/%d near-duplicate chunks for %s", skipped, len(chunks), doc.FileName)
		}
		chunks = filteredChunks
		vectors = filteredVectors

		if len(chunks) == 0 {
			log.Printf("All chunks were duplicates for %s, nothing to insert", doc.FileName)
			return nil
		}
	}

	return wp.batchInsert(ctx, notebookID, chunks, vectors)
}

// batchInsert inserts chunks with their vectors into Weaviate in batches.
func (wp *WorkerPool) batchInsert(ctx context.Context, notebookID string, chunks []Chunk, vectors [][]float32) error {
	const batchSize = 100

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		var objects []*models.Object
		for j := i; j < end; j++ {
			chunk := chunks[j]
			obj := &models.Object{
				Class: db.ClassName,
				Properties: map[string]interface{}{
					"notebook_id":    notebookID,
					"content":        chunk.Text,
					"file_name":      chunk.FileName,
					"page_number":    chunk.PageNumber,
					"chunk_index":    chunk.ChunkIndex,
					"header_context": chunk.HeaderContext,
				},
				Vector: vectors[j],
			}
			objects = append(objects, obj)
		}

		_, err := wp.weaviateClient.Batch().ObjectsBatcher().
			WithObjects(objects...).
			WithConsistencyLevel(replication.ConsistencyLevel.ONE).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("batch insert failed at offset %d: %w", i, err)
		}
	}

	log.Printf("✓ Inserted %d chunks for notebook %s", len(chunks), notebookID)
	return nil
}
