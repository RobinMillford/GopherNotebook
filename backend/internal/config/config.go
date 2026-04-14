package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration.
type Config struct {
	// Server
	ServerPort string

	// LocalAI (Embedding)
	LocalAIURL       string
	EmbeddingModel   string
	EmbeddingDim     int

	// Reranker (llama-server)
	RerankerURL   string
	RerankerModel string
	RerankerTopN  int

	// Weaviate
	WeaviateHost   string
	WeaviateScheme string

	// Notebook storage
	NotebookDataDir string

	// Upload
	UploadDir string

	// Semantic deduplication (cosine distance threshold; 0 = disabled)
	SemanticDedupThreshold float64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort:     envOrDefault("SERVER_PORT", "8090"),
		LocalAIURL:     envOrDefault("LOCALAI_URL", "http://localhost:8081"),
		EmbeddingModel: envOrDefault("EMBEDDING_MODEL", "qwen3-embed"),
		EmbeddingDim:   envOrDefaultInt("EMBEDDING_DIM", 1024),
		RerankerURL:    envOrDefault("RERANKER_URL", "http://localhost:8082"),
		RerankerModel:  envOrDefault("RERANKER_MODEL", "Qwen3-Reranker-0.6B"),
		RerankerTopN:   envOrDefaultInt("RERANKER_TOP_N", 5),
		WeaviateHost:   envOrDefault("WEAVIATE_HOST", "localhost:8080"),
		WeaviateScheme: envOrDefault("WEAVIATE_SCHEME", "http"),
		NotebookDataDir:        envOrDefault("NOTEBOOK_DATA_DIR", "./data/notebooks"),
		UploadDir:              envOrDefault("UPLOAD_DIR", "./data/uploads"),
		SemanticDedupThreshold: envOrDefaultFloat("SEMANTIC_DEDUP_THRESHOLD", 0.03),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefaultFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
