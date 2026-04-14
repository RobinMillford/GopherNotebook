package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yamin/gophernotebook/internal/ingest"
	"github.com/yamin/gophernotebook/internal/notebook"
)

// maxUploadFileSize is the per-file size cap enforced before streaming to disk.
const maxUploadFileSize = 50 << 20 // 50 MB

// Active ingestion progress channels, keyed by notebook ID.
var (
	progressChannels = make(map[string][]chan ingest.IngestProgress)
	progressMu       sync.RWMutex
)

// UploadFiles handles multipart file upload and triggers async ingestion.
func (s *Server) UploadFiles(c *gin.Context) {
	notebookID := c.Param("id")
	if !isValidID(notebookID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	// Verify notebook exists
	if _, err := s.nbManager.Get(notebookID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notebook not found"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
		return
	}

	// Create upload directory for this notebook
	uploadDir := filepath.Join(s.cfg.UploadDir, notebookID)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Save files to disk (streaming, not loading all into RAM)
	// Load existing sources once so deduplication check below is O(n) not O(n²).
	existingNotebook, _ := s.nbManager.Get(notebookID)

	var jobs []ingest.FileJob
	for _, fileHeader := range files {
		// filepath.Base prevents path traversal (e.g. "../../etc/passwd").
		safeFilename := filepath.Base(fileHeader.Filename)
		destPath := filepath.Join(uploadDir, safeFilename)

		if fileHeader.Size > maxUploadFileSize {
			log.Printf("Rejecting oversized file %s (%d bytes, limit %d)", safeFilename, fileHeader.Size, maxUploadFileSize)
			continue
		}

		// Skip files already successfully ingested to prevent duplicate Weaviate chunks.
		if existingNotebook != nil && isAlreadyIngested(existingNotebook.Sources, safeFilename) {
			log.Printf("Skipping already-ingested file: %s", safeFilename)
			continue
		}

		// Stream file to disk
		src, err := fileHeader.Open()
		if err != nil {
			log.Printf("Failed to open uploaded file %s: %v", fileHeader.Filename, err)
			continue
		}

		dst, err := os.Create(destPath)
		if err != nil {
			src.Close()
			log.Printf("Failed to create file %s: %v", destPath, err)
			continue
		}

		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			src.Close()
			log.Printf("Failed to write file %s: %v", destPath, err)
			continue
		}
		dst.Close()
		src.Close()

		// Check if file type is supported
		ext := filepath.Ext(safeFilename)
		if !ingest.SupportedExtensions[ext] {
			log.Printf("Skipping unsupported file type: %s", safeFilename)
			continue
		}

		jobs = append(jobs, ingest.FileJob{
			FilePath:   destPath,
			NotebookID: notebookID,
		})

		// Record source as "processing"
		_ = s.nbManager.AddSource(notebookID, notebook.Source{
			FileName:   safeFilename,
			FileSize:   fileHeader.Size,
			IngestedAt: time.Now(),
			Status:     "processing",
		})
	}

	if len(jobs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No supported files found"})
		return
	}

	// Start async ingestion
	go func() {
		// Use a detached background context since this outlives the HTTP request.
		ctx := context.Background()

		s.workerPool.Ingest(ctx, jobs, func(progress ingest.IngestProgress) {
			// Update source status
			status := "ingested"
			errMsg := ""
			if progress.Error != "" {
				status = "failed"
				errMsg = progress.Error
			}

			fileName := filepath.Base(progress.CurrentFile)
			_ = s.nbManager.AddSource(notebookID, notebook.Source{
				FileName:   fileName,
				IngestedAt: time.Now(),
				Status:     status,
				Error:      errMsg,
			})

			// Broadcast progress to SSE listeners
			broadcastProgress(notebookID, progress)
		})
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":    fmt.Sprintf("Ingestion started for %d files", len(jobs)),
		"totalFiles": len(jobs),
	})
}

// IngestProgress streams ingestion progress via SSE.
func (s *Server) IngestProgress(c *gin.Context) {
	notebookID := c.Param("id")
	if !isValidID(notebookID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Create a channel for this client
	ch := make(chan ingest.IngestProgress, 50)

	progressMu.Lock()
	progressChannels[notebookID] = append(progressChannels[notebookID], ch)
	progressMu.Unlock()

	defer func() {
		progressMu.Lock()
		channels := progressChannels[notebookID]
		for i, c := range channels {
			if c == ch {
				progressChannels[notebookID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		progressMu.Unlock()
		close(ch)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case progress, ok := <-ch:
			if !ok {
				return false
			}
			data, _ := json.Marshal(progress)
			c.SSEvent("progress", string(data))
			c.Writer.Flush()

			return progress.Status != "done"
		case <-c.Request.Context().Done():
			return false
		case <-time.After(30 * time.Second):
			// Keepalive
			c.SSEvent("ping", "keepalive")
			c.Writer.Flush()
			return true
		}
	})
}

// isAlreadyIngested reports whether a file was previously ingested successfully.
func isAlreadyIngested(sources []notebook.Source, fileName string) bool {
	for _, s := range sources {
		if s.FileName == fileName && s.Status == "ingested" {
			return true
		}
	}
	return false
}

// broadcastProgress sends progress to all SSE listeners for a notebook.
func broadcastProgress(notebookID string, progress ingest.IngestProgress) {
	progressMu.RLock()
	defer progressMu.RUnlock()

	for _, ch := range progressChannels[notebookID] {
		select {
		case ch <- progress:
		default:
			// Drop if channel is full (slow consumer)
		}
	}
}
