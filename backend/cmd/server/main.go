package main

import (
	"context"
	"log"

	"github.com/yamin/gophernotebook/internal/api"
	"github.com/yamin/gophernotebook/internal/config"
	"github.com/yamin/gophernotebook/internal/db"
	"github.com/yamin/gophernotebook/internal/ingest"
	"github.com/yamin/gophernotebook/internal/notebook"
)

func main() {
	log.Println("🚀 Starting GopherNotebook Backend...")

	// Load configuration
	cfg := config.Load()

	// Initialize Weaviate client
	weaviateClient, err := db.NewWeaviateClient(cfg.WeaviateHost, cfg.WeaviateScheme)
	if err != nil {
		log.Fatalf("Failed to connect to Weaviate: %v", err)
	}

	// Ensure schema exists
	if err := db.EnsureSchema(context.Background(), weaviateClient); err != nil {
		log.Fatalf("Failed to initialize Weaviate schema: %v", err)
	}

	// Initialize embedder
	embedder := ingest.NewEmbedder(cfg.LocalAIURL, cfg.EmbeddingModel)

	// Initialize notebook manager
	nbManager, err := notebook.NewManager(cfg.NotebookDataDir)
	if err != nil {
		log.Fatalf("Failed to initialize notebook manager: %v", err)
	}

	// Create and start the API server
	server := api.NewServer(cfg, nbManager, weaviateClient, embedder)

	log.Printf("✓ GopherNotebook Backend listening on :%s", cfg.ServerPort)
	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
