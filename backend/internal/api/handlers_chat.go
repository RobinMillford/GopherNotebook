package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gin-gonic/gin"
	"github.com/yamin/gophernotebook/internal/generate"
	"github.com/yamin/gophernotebook/internal/notebook"
)
// ChatRequest is the JSON body for a chat request.
type ChatRequest struct {
	Query string `json:"query" binding:"required"`
}

// Chat performs two-stage retrieval (hybrid + rerank) and streams the LLM response via SSE.
func (s *Server) Chat(c *gin.Context) {
	notebookID := c.Param("id")

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query is required"})
		return
	}

	// Extract LLM config from headers
	apiKey := c.GetHeader("X-API-Key")
	provider := c.GetHeader("X-LLM-Provider")
	model := c.GetHeader("X-LLM-Model")

	if apiKey == "" || provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-API-Key and X-LLM-Provider headers are required"})
		return
	}

	if model == "" {
		// Default models per provider
		switch provider {
		case "openai":
			model = "gpt-4o"
		case "google", "gemini":
			model = "gemini-2.0-flash"
		case "anthropic":
			model = "claude-3-7-sonnet-20250219"
		}
	}

	ctx := c.Request.Context()

	// Stage 1: Hybrid search — fetch top 20
	hybridResults, err := s.searcher.Search(ctx, req.Query, notebookID, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed: " + err.Error()})
		return
	}

	if len(hybridResults) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":   "No relevant documents found in this notebook.",
			"citations": []interface{}{},
		})
		return
	}

	// Stage 2: Rerank — get top N
	rerankedResults, err := s.reranker.Rerank(req.Query, hybridResults, s.cfg.RerankerTopN)
	if err != nil {
		// Fallback to hybrid results if reranker fails
		rerankedResults = hybridResults
		if len(rerankedResults) > s.cfg.RerankerTopN {
			rerankedResults = rerankedResults[:s.cfg.RerankerTopN]
		}
	}

	// Load history
	messages, _ := s.nbManager.GetMessages(notebookID)

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Save user message to history
	userMsg := notebook.Message{
		ID:        "u_" + uuid.New().String(),
		Role:      "user",
		Content:   req.Query,
		CreatedAt: time.Now(),
	}
	_ = s.nbManager.SaveMessage(notebookID, userMsg)

	// Stream the LLM response
	c.Stream(func(w io.Writer) bool {
		var fullResponse string
		var finalCitations interface{}

		err := s.generator.GenerateStream(ctx, req.Query, rerankedResults, provider, apiKey, model, messages,
			func(chunk generate.StreamChunk) {
				fullResponse += chunk.Content
				if len(chunk.Citations) > 0 {
					finalCitations = chunk.Citations
				}
				data, _ := json.Marshal(chunk)
				c.SSEvent("message", string(data))
				c.Writer.Flush()
			},
		)
		if err != nil {
			errChunk := generate.StreamChunk{
				Content: "\n\n[Error: " + err.Error() + "]",
				Done:    true,
			}
			fullResponse += errChunk.Content
			data, _ := json.Marshal(errChunk)
			c.SSEvent("message", string(data))
			c.Writer.Flush()
		}

		// Save assistant message to history
		asstMsg := notebook.Message{
			ID:        "a_" + uuid.New().String(),
			Role:      "assistant",
			Content:   fullResponse,
			Citations: finalCitations,
			CreatedAt: time.Now(),
		}
		_ = s.nbManager.SaveMessage(notebookID, asstMsg)

		return false
	})

	_ = time.Now() // Prevent unused import
}
