package ingest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Embedder handles batch embedding requests to LocalAI.
type Embedder struct {
	baseURL   string
	model     string
	batchSize int
	client    *http.Client
}

// embeddingRequest is the OpenAI-compatible embedding request body.
type embeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// embeddingResponse is the OpenAI-compatible embedding response.
type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// NewEmbedder creates an embedder that calls LocalAI's /v1/embeddings endpoint.
func NewEmbedder(baseURL, model string) *Embedder {
	return &Embedder{
		baseURL:   baseURL,
		model:     model,
		batchSize: 32,
		client:    &http.Client{},
	}
}

// EmbedBatch generates embeddings for a list of texts in batches.
// Returns vectors in the same order as the input texts.
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += e.batchSize {
		end := i + e.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		embeddings, err := e.embedSingle(batch)
		if err != nil {
			return nil, fmt.Errorf("batch embed failed at offset %d: %w", i, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// EmbedQuery generates a single embedding for a search query.
func (e *Embedder) EmbedQuery(text string) ([]float32, error) {
	embeddings, err := e.embedSingle([]string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return embeddings[0], nil
}

// embedSingle calls LocalAI /v1/embeddings for a batch of texts.
func (e *Embedder) embedSingle(texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Input: texts,
		Model: e.model,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	resp, err := e.client.Post(
		e.baseURL+"/v1/embeddings",
		"application/json",
		bytes.NewReader(jsonBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API returned %d: %s", resp.StatusCode, string(body))
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	// Sort by index to ensure correct order
	results := make([][]float32, len(embResp.Data))
	for _, d := range embResp.Data {
		results[d.Index] = d.Embedding
	}

	return results, nil
}
