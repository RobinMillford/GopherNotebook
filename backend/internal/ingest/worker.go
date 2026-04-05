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
	weaviateClient *weaviate.Client
	embedder       *Embedder
	numWorkers     int
}

// NewWorkerPool creates a worker pool with NumCPU-1 workers (minimum 1).
func NewWorkerPool(weaviateClient *weaviate.Client, embedder *Embedder) *WorkerPool {
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	log.Printf("Worker pool initialized with %d workers", workers)

	return &WorkerPool{
		weaviateClient: weaviateClient,
		embedder:       embedder,
		numWorkers:     workers,
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

// processFile is the per-file ingestion pipeline:
// Parse → Chunk → Embed → Batch Insert to Weaviate
func (wp *WorkerPool) processFile(ctx context.Context, job FileJob) error {
	// Step 1: Parse the document
	doc, err := ParseFile(job.FilePath)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	// Step 2: Chunk the document
	chunks := ChunkDocument(doc)
	if len(chunks) == 0 {
		return fmt.Errorf("no chunks produced from %s", job.FilePath)
	}

	// Step 3: Extract texts for embedding
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}

	// Step 4: Batch embed
	vectors, err := wp.embedder.EmbedBatch(texts)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	if len(vectors) != len(chunks) {
		return fmt.Errorf("vector count mismatch: got %d vectors for %d chunks", len(vectors), len(chunks))
	}

	// Step 5: Batch insert to Weaviate
	return wp.batchInsert(ctx, job.NotebookID, chunks, vectors)
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
