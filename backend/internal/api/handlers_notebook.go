package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yamin/gophernotebook/internal/db"
	"github.com/yamin/gophernotebook/internal/notebook"
)

// CreateNotebookRequest is the JSON body for creating a notebook.
type CreateNotebookRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// ListNotebooks returns all notebooks.
func (s *Server) ListNotebooks(c *gin.Context) {
	notebooks, err := s.nbManager.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if notebooks == nil {
		notebooks = []notebook.Notebook{}
	}
	c.JSON(http.StatusOK, notebooks)
}

// CreateNotebook creates a new notebook.
func (s *Server) CreateNotebook(c *gin.Context) {
	var req CreateNotebookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	nb, err := s.nbManager.Create(req.Name, req.Description, req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, nb)
}

// GetNotebook returns a notebook with its sources.
func (s *Server) GetNotebook(c *gin.Context) {
	id := c.Param("id")
	if !isValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	detail, err := s.nbManager.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notebook not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// DeleteNotebook removes a notebook and all its chunks from Weaviate.
func (s *Server) DeleteNotebook(c *gin.Context) {
	id := c.Param("id")
	if !isValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	// Delete chunks from Weaviate first
	if err := db.DeleteNotebookChunks(c.Request.Context(), s.weaviate, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chunks: " + err.Error()})
		return
	}

	// Delete notebook metadata
	if err := s.nbManager.Delete(id); err != nil {
		// Weaviate chunks were already deleted above. Log so the orphan state is visible.
		log.Printf("WARNING: notebook %s Weaviate chunks deleted but metadata removal failed: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Notebook partially deleted; retry to clean up"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notebook deleted"})
}

// UpdateNotebookRequest holds optional fields for PATCH /notebooks/:id.
type UpdateNotebookRequest struct {
	Name         *string   `json:"name"`
	Description  *string   `json:"description"`
	SystemPrompt *string   `json:"systemPrompt"`
	Tags         *[]string `json:"tags"`
}

// UpdateNotebook applies a partial update to a notebook.
func (s *Server) UpdateNotebook(c *gin.Context) {
	id := c.Param("id")
	if !isValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	var req UpdateNotebookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nb, err := s.nbManager.UpdateNotebook(id, notebook.NotebookUpdate{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Tags:         req.Tags,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, nb)
}

// ListSources returns sources for a notebook.
func (s *Server) ListSources(c *gin.Context) {
	id := c.Param("id")
	if !isValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	sources, err := s.nbManager.GetSources(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notebook not found"})
		return
	}

	c.JSON(http.StatusOK, sources)
}

// DeleteSource removes a source file's chunks from Weaviate and its metadata.
func (s *Server) DeleteSource(c *gin.Context) {
	notebookID := c.Param("id")
	if !isValidID(notebookID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}
	fileName := c.Param("filename")

	if err := db.DeleteSourceChunks(c.Request.Context(), s.weaviate, notebookID, fileName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chunks: " + err.Error()})
		return
	}

	if err := s.nbManager.RemoveSource(notebookID, fileName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Source deleted"})
}

// ClearChatHistory deletes the chat history for a notebook.
func (s *Server) ClearChatHistory(c *gin.Context) {
	id := c.Param("id")
	if !isValidID(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	if err := s.nbManager.ClearChatHistory(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear chat history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat history cleared"})
}
