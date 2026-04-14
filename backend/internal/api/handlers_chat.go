package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/yamin/gophernotebook/internal/generate"
	"github.com/yamin/gophernotebook/internal/notebook"
)
// ChatRequest is the JSON body for a chat request.
type ChatRequest struct {
	Query          string   `json:"query" binding:"required"`
	RetrievalLimit *int     `json:"retrievalLimit"`
	RerankerTopN   *int     `json:"rerankerTopN"`
	Temperature    *float64 `json:"temperature"`
	SourceFilter   []string `json:"sourceFilter"`
	HyDE           bool     `json:"hyde"`
}

// TruncateRequest is the JSON body for truncating message history.
type TruncateRequest struct {
	MessageID string `json:"messageID" binding:"required"`
}

// TruncateMessages removes a message and all subsequent messages from chat history.
func (s *Server) TruncateMessages(c *gin.Context) {
	notebookID := c.Param("id")
	if !isValidID(notebookID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
		return
	}

	var req TruncateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messageID is required"})
		return
	}

	if err := s.nbManager.TruncateMessagesFrom(notebookID, req.MessageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Messages truncated"})
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

	if !isValidID(notebookID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notebook ID"})
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

	// Apply per-request retrieval params with bounds-checked defaults.
	retrievalLimit := 20
	if req.RetrievalLimit != nil && *req.RetrievalLimit > 0 && *req.RetrievalLimit <= 50 {
		retrievalLimit = *req.RetrievalLimit
	}

	rerankerTopN := s.cfg.RerankerTopN
	if req.RerankerTopN != nil && *req.RerankerTopN > 0 && *req.RerankerTopN <= retrievalLimit {
		rerankerTopN = *req.RerankerTopN
	}

	temperature := 0.3
	if req.Temperature != nil && *req.Temperature >= 0 && *req.Temperature <= 2 {
		temperature = *req.Temperature
	}

	// Load notebook system prompt (best-effort; empty string if notebook unavailable).
	systemPrompt := ""
	if detail, err := s.nbManager.Get(notebookID); err == nil {
		systemPrompt = detail.SystemPrompt
	}

	// HyDE: replace the search query with a hypothetical answer to improve recall.
	searchQuery := req.Query
	if req.HyDE {
		if hydeDoc, hydeErr := s.generator.HyDEGenerate(ctx, req.Query, provider, apiKey, model); hydeErr == nil {
			searchQuery = hydeDoc
		} else {
			log.Printf("HyDE generation failed, using original query: %v", hydeErr)
		}
	}

	// Stage 1: Hybrid search
	hybridResults, err := s.searcher.Search(ctx, searchQuery, notebookID, retrievalLimit, req.SourceFilter)
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
	rerankedResults, err := s.reranker.Rerank(req.Query, hybridResults, rerankerTopN)
	if err != nil {
		// Fallback to hybrid results if reranker fails; log so infra issues are visible.
		log.Printf("reranker unavailable, falling back to hybrid results: %v", err)
		rerankedResults = hybridResults
		if len(rerankedResults) > rerankerTopN {
			rerankedResults = rerankedResults[:rerankerTopN]
		}
	}

	// Load history — cap to the most recent 50 messages to bound prompt size.
	messages, _ := s.nbManager.GetMessages(notebookID)
	const historyWindow = 50
	if len(messages) > historyWindow {
		messages = messages[len(messages)-historyWindow:]
	}

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

		err := s.generator.GenerateStream(ctx, req.Query, rerankedResults, provider, apiKey, model, messages, temperature, systemPrompt,
			func(chunk generate.StreamChunk) {
				// Stop writing if the client already disconnected.
				select {
				case <-ctx.Done():
					return
				default:
				}
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
				Content: "\n\n[Error: " + sanitizeError(err, apiKey) + "]",
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
}
