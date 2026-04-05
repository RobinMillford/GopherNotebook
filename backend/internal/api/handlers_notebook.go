package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yamin/gophernotebook/internal/db"
	"github.com/yamin/gophernotebook/internal/notebook"
)

// CreateNotebookRequest is the JSON body for creating a notebook.
type CreateNotebookRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
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

	nb, err := s.nbManager.Create(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, nb)
}

// GetNotebook returns a notebook with its sources.
func (s *Server) GetNotebook(c *gin.Context) {
	id := c.Param("id")

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

	// Delete chunks from Weaviate first
	if err := db.DeleteNotebookChunks(c.Request.Context(), s.weaviate, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chunks: " + err.Error()})
		return
	}

	// Delete notebook metadata
	if err := s.nbManager.Delete(id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notebook not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notebook deleted"})
}

// ListSources returns sources for a notebook.
func (s *Server) ListSources(c *gin.Context) {
	id := c.Param("id")

	sources, err := s.nbManager.GetSources(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notebook not found"})
		return
	}

	c.JSON(http.StatusOK, sources)
}

// DeleteSource removes a source file's chunks from Weaviate.
func (s *Server) DeleteSource(c *gin.Context) {
	notebookID := c.Param("id")
	fileName := c.Param("filename")

	// TODO: Delete specific file's chunks from Weaviate (needs batch delete with compound filter)
	// For now, remove from metadata
	if err := s.nbManager.RemoveSource(notebookID, fileName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Source deleted"})
}
