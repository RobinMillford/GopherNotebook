package retrieve

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/yamin/gophernotebook/internal/db"
	"github.com/yamin/gophernotebook/internal/ingest"
)

// RetrievedChunk is a chunk returned from search, with a relevance score.
type RetrievedChunk struct {
	Content       string  `json:"content"`
	FileName      string  `json:"fileName"`
	PageNumber    int     `json:"pageNumber"`
	ChunkIndex    int     `json:"chunkIndex"`
	HeaderContext string  `json:"headerContext"`
	Score         float64 `json:"score"`
}

// HybridSearcher performs hybrid (vector + BM25) search on Weaviate.
type HybridSearcher struct {
	client   *weaviate.Client
	embedder *ingest.Embedder
}

// NewHybridSearcher creates a searcher that uses Weaviate hybrid search.
func NewHybridSearcher(client *weaviate.Client, embedder *ingest.Embedder) *HybridSearcher {
	return &HybridSearcher{
		client:   client,
		embedder: embedder,
	}
}

// Search performs a hybrid search filtered by notebook_id.
// Returns the top `limit` results ranked by hybrid score (alpha=0.5 = balanced).
func (hs *HybridSearcher) Search(ctx context.Context, query string, notebookID string, limit int) ([]RetrievedChunk, error) {
	// Generate query embedding
	queryVector, err := hs.embedder.EmbedQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Build the notebook_id filter
	whereFilter := filters.Where().
		WithPath([]string{"notebook_id"}).
		WithOperator(filters.Equal).
		WithValueString(notebookID)

	// Build hybrid search with vector
	hybridArg := hs.client.GraphQL().HybridArgumentBuilder().
		WithQuery(query).
		WithVector(queryVector).
		WithAlpha(0.5) // 0.5 = equal weight to BM25 and vector

	// Define fields to return
	fields := []graphql.Field{
		{Name: "content"},
		{Name: "file_name"},
		{Name: "page_number"},
		{Name: "chunk_index"},
		{Name: "header_context"},
		{Name: "_additional { score }"},
	}

	result, err := hs.client.GraphQL().Get().
		WithClassName(db.ClassName).
		WithFields(fields...).
		WithHybrid(hybridArg).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	return parseSearchResults(result)
}

// parseSearchResults extracts RetrievedChunk objects from the GraphQL response.
func parseSearchResults(result *models.GraphQLResponse) ([]RetrievedChunk, error) {
	if result == nil || result.Data == nil {
		return nil, fmt.Errorf("empty search result")
	}

	getData, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response structure")
	}

	classData, ok := getData[db.ClassName].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no results for class %s", db.ClassName)
	}

	var chunks []RetrievedChunk
	for _, item := range classData {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		chunk := RetrievedChunk{
			Content:       getStringField(obj, "content"),
			FileName:      getStringField(obj, "file_name"),
			HeaderContext: getStringField(obj, "header_context"),
			PageNumber:    getIntField(obj, "page_number"),
			ChunkIndex:    getIntField(obj, "chunk_index"),
		}

		// Extract score from _additional
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if score, ok := additional["score"].(float64); ok {
				chunk.Score = score
			}
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

func getStringField(obj map[string]interface{}, key string) string {
	if v, ok := obj[key].(string); ok {
		return v
	}
	return ""
}

func getIntField(obj map[string]interface{}, key string) int {
	if v, ok := obj[key].(float64); ok {
		return int(v)
	}
	return 0
}
