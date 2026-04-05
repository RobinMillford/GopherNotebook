package notebook

import (
	"time"
)

// Notebook represents a workspace that contains documents and chat history.
type Notebook struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	FileCount   int       `json:"fileCount"`
}

// Source represents a file that has been ingested into a notebook.
type Source struct {
	FileName   string    `json:"fileName"`
	FileSize   int64     `json:"fileSize"`
	ChunkCount int       `json:"chunkCount"`
	IngestedAt time.Time `json:"ingestedAt"`
	Status     string    `json:"status"` // "ingested", "failed", "processing"
	Error      string    `json:"error,omitempty"`
}

// NotebookDetail is a Notebook plus its list of sources.
type NotebookDetail struct {
	Notebook
	Sources []Source `json:"sources"`
}
