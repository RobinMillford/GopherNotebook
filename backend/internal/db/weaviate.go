package db

import (
	"context"
	"fmt"
	"log"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

// NewWeaviateClient creates and returns a connected Weaviate client.
func NewWeaviateClient(host, scheme string) (*weaviate.Client, error) {
	cfg := weaviate.Config{
		Host:   host,
		Scheme: scheme,
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create weaviate client: %w", err)
	}

	// Verify connectivity
	ready, err := client.Misc().ReadyChecker().Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("weaviate readiness check failed: %w", err)
	}
	if !ready {
		return nil, fmt.Errorf("weaviate is not ready")
	}

	log.Println("✓ Connected to Weaviate at", scheme+"://"+host)
	return client, nil
}
