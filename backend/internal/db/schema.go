package db

import (
	"context"
	"fmt"
	"log"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/weaviate/weaviate/entities/schema"
)

const ClassName = "DocumentChunk"

// EnsureSchema creates the DocumentChunk class in Weaviate if it doesn't exist.
// The class uses vectorizer "none" (BYO vectors from LocalAI) and is configured
// for hybrid search (BM25 + vector).
func EnsureSchema(ctx context.Context, client *weaviate.Client) error {
	// Check if class already exists
	existingSchema, err := client.Schema().Getter().Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	for _, class := range existingSchema.Classes {
		if class.Class == ClassName {
			log.Printf("✓ Weaviate class '%s' already exists", ClassName)
			return nil
		}
	}

	// Define the DocumentChunk class
	classObj := &models.Class{
		Class:       ClassName,
		Description: "A chunk of a document belonging to a notebook",
		Vectorizer:  "none", // We provide our own vectors via LocalAI
		VectorIndexType: "hnsw",
		VectorIndexConfig: map[string]interface{}{
			"distance": "cosine",
		},
		// Enable BM25 inverted index for hybrid search
		InvertedIndexConfig: &models.InvertedIndexConfig{
			Bm25: &models.BM25Config{
				B:  0.75,
				K1: 1.2,
			},
		},
		Properties: []*models.Property{
			{
				Name:         "notebook_id",
				Description:  "UUID of the notebook this chunk belongs to",
				DataType:     schema.DataTypeText.PropString(),
				Tokenization: "field", // Exact match for filtering
			},
			{
				Name:         "content",
				Description:  "The text content of the chunk",
				DataType:     schema.DataTypeText.PropString(),
				Tokenization: "word", // Word-level tokenization for BM25
			},
			{
				Name:         "file_name",
				Description:  "Original filename of the source document",
				DataType:     schema.DataTypeText.PropString(),
				Tokenization: "field",
			},
			{
				Name:         "page_number",
				Description:  "Page number in the source document (0 if N/A)",
				DataType:     schema.DataTypeInt.PropString(),
			},
			{
				Name:         "chunk_index",
				Description:  "Sequential index of this chunk within the file",
				DataType:     schema.DataTypeInt.PropString(),
			},
			{
				Name:         "header_context",
				Description:  "Contextual header: section title or filename prefix",
				DataType:     schema.DataTypeText.PropString(),
				Tokenization: "word",
			},
		},
	}

	err = client.Schema().ClassCreator().WithClass(classObj).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to create class '%s': %w", ClassName, err)
	}

	log.Printf("✓ Created Weaviate class '%s'", ClassName)
	return nil
}

// DeleteSourceChunks removes all DocumentChunk objects for a specific file within a notebook.
func DeleteSourceChunks(ctx context.Context, client *weaviate.Client, notebookID, fileName string) error {
	whereFilter := filters.Where().
		WithOperator(filters.And).
		WithOperands([]*filters.WhereBuilder{
			filters.Where().WithPath([]string{"notebook_id"}).WithOperator(filters.Equal).WithValueString(notebookID),
			filters.Where().WithPath([]string{"file_name"}).WithOperator(filters.Equal).WithValueString(fileName),
		})

	result, err := client.Batch().ObjectsBatchDeleter().
		WithClassName(ClassName).
		WithWhere(whereFilter).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete chunks for %s in notebook %s: %w", fileName, notebookID, err)
	}

	if result != nil && result.Results != nil {
		log.Printf("✓ Deleted %d chunks for file %s in notebook %s", result.Results.Successful, fileName, notebookID)
	}
	return nil
}

// IsNearDuplicate returns true if any existing chunk in the notebook is within
// distanceThreshold cosine distance of the provided vector.
func IsNearDuplicate(ctx context.Context, client *weaviate.Client, notebookID string, vector []float32, distanceThreshold float32) (bool, error) {
	whereFilter := filters.Where().
		WithPath([]string{"notebook_id"}).
		WithOperator(filters.Equal).
		WithValueString(notebookID)

	nearVector := client.GraphQL().NearVectorArgBuilder().
		WithVector(vector).
		WithDistance(distanceThreshold)

	fields := []graphql.Field{
		{Name: "_additional { distance }"},
	}

	result, err := client.GraphQL().Get().
		WithClassName(ClassName).
		WithFields(fields...).
		WithNearVector(nearVector).
		WithWhere(whereFilter).
		WithLimit(1).
		Do(ctx)
	if err != nil {
		return false, fmt.Errorf("near-duplicate query failed: %w", err)
	}

	if result == nil || result.Data == nil {
		return false, nil
	}

	getData, ok := result.Data["Get"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	classData, ok := getData[ClassName].([]interface{})
	return ok && len(classData) > 0, nil
}

// DeleteNotebookChunks removes all DocumentChunk objects for a given notebook_id.
func DeleteNotebookChunks(ctx context.Context, client *weaviate.Client, notebookID string) error {
	// Use batch delete with a where filter
	whereFilter := filters.Where().
		WithPath([]string{"notebook_id"}).
		WithOperator(filters.Equal).
		WithValueString(notebookID)

	result, err := client.Batch().ObjectsBatchDeleter().
		WithClassName(ClassName).
		WithWhere(whereFilter).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete chunks for notebook %s: %w", notebookID, err)
	}

	if result != nil && result.Results != nil {
		log.Printf("✓ Deleted %d chunks for notebook %s", result.Results.Successful, notebookID)
	}
	return nil
}
