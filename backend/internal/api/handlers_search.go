package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GlobalSearchRequest is the JSON body for POST /api/search.
type GlobalSearchRequest struct {
	Query       string   `json:"query" binding:"required"`
	NotebookIDs []string `json:"notebookIDs"` // empty = all notebooks
	Limit       int      `json:"limit"`        // default 10, max 50
}

// GlobalSearch retrieves matching chunks across all (or specified) notebooks.
// No LLM call — purely embedding + Weaviate hybrid search.
func (s *Server) GlobalSearch(c *gin.Context) {
	var req GlobalSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	chunks, err := s.searcher.SearchGlobal(c.Request.Context(), req.Query, req.NotebookIDs, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if chunks == nil {
		chunks = nil // keep nil — JSON encodes as null, which is fine for frontend
	}

	c.JSON(http.StatusOK, chunks)
}
