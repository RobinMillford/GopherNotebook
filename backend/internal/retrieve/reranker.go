package retrieve

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// Reranker calls the llama-server /v1/rerank endpoint to rerank retrieved chunks.
type Reranker struct {
	baseURL string
	model   string
	client  *http.Client
}

// rerankRequest matches the llama.cpp server /v1/rerank request format.
type rerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

// rerankResponse matches the llama.cpp server /v1/rerank response format.
type rerankResponse struct {
	Results []rerankResult `json:"results"`
}

type rerankResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

// NewReranker creates a reranker that calls llama-server's /v1/rerank endpoint.
func NewReranker(baseURL, model string) *Reranker {
	return &Reranker{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Rerank takes the query and a list of retrieved chunks, sends them to the
// reranker model, and returns the chunks sorted by relevance score.
func (r *Reranker) Rerank(query string, chunks []RetrievedChunk, topN int) ([]RetrievedChunk, error) {
	if len(chunks) == 0 {
		return chunks, nil
	}

	// Extract document texts
	documents := make([]string, len(chunks))
	for i, c := range chunks {
		documents[i] = c.Content
	}

	reqBody := rerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
		TopN:      topN,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rerank request: %w", err)
	}

	resp, err := r.client.Post(
		r.baseURL+"/v1/rerank",
		"application/json",
		bytes.NewReader(jsonBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("rerank request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rerank API returned %d: %s", resp.StatusCode, string(body))
	}

	var rerankResp rerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, fmt.Errorf("failed to decode rerank response: %w", err)
	}

	// Sort results by relevance score descending
	sort.Slice(rerankResp.Results, func(i, j int) bool {
		return rerankResp.Results[i].RelevanceScore > rerankResp.Results[j].RelevanceScore
	})

	// Map back to chunks with updated scores
	var reranked []RetrievedChunk
	for _, result := range rerankResp.Results {
		if result.Index < 0 || result.Index >= len(chunks) {
			continue
		}
		chunk := chunks[result.Index]
		chunk.Score = result.RelevanceScore
		reranked = append(reranked, chunk)
	}

	// Limit to topN
	if topN > 0 && len(reranked) > topN {
		reranked = reranked[:topN]
	}

	return reranked, nil
}
