package api

import (
	"github.com/gin-gonic/gin"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/yamin/gophernotebook/internal/config"
	"github.com/yamin/gophernotebook/internal/generate"
	"github.com/yamin/gophernotebook/internal/ingest"
	"github.com/yamin/gophernotebook/internal/notebook"
	"github.com/yamin/gophernotebook/internal/retrieve"
)

// Server holds all the dependencies for the API handlers.
type Server struct {
	cfg        *config.Config
	router     *gin.Engine
	nbManager  *notebook.Manager
	workerPool *ingest.WorkerPool
	embedder   *ingest.Embedder
	searcher   *retrieve.HybridSearcher
	reranker   *retrieve.Reranker
	generator  *generate.Generator
	weaviate   *weaviate.Client
}

// NewServer creates a new API server with all dependencies wired up.
func NewServer(
	cfg *config.Config,
	nbManager *notebook.Manager,
	weaviateClient *weaviate.Client,
	embedder *ingest.Embedder,
) *Server {
	s := &Server{
		cfg:        cfg,
		nbManager:  nbManager,
		weaviate:   weaviateClient,
		embedder:   embedder,
		workerPool: ingest.NewWorkerPool(weaviateClient, embedder, float32(cfg.SemanticDedupThreshold)),
		searcher:   retrieve.NewHybridSearcher(weaviateClient, embedder),
		reranker:   retrieve.NewReranker(cfg.RerankerURL, cfg.RerankerModel),
		generator:  generate.NewGenerator(),
	}

	s.setupRouter()
	return s
}

// setupRouter configures all routes.
func (s *Server) setupRouter() {
	r := gin.Default()
	r.Use(CORSMiddleware())

	// Set max multipart memory for file uploads (512 MB for large batches)
	r.MaxMultipartMemory = 512 << 20

	api := r.Group("/api")
	{
		// Notebook CRUD
		api.GET("/notebooks", s.ListNotebooks)
		api.POST("/notebooks", s.CreateNotebook)
		api.GET("/notebooks/:id", s.GetNotebook)
		api.PATCH("/notebooks/:id", s.UpdateNotebook)
		api.DELETE("/notebooks/:id", s.DeleteNotebook)

		// Sources
		api.GET("/notebooks/:id/sources", s.ListSources)
		api.DELETE("/notebooks/:id/sources/:filename", s.DeleteSource)
		api.POST("/notebooks/:id/sources/:filename/reingest", s.ReIngestSource)

		// Ingestion
		api.POST("/notebooks/:id/upload", s.UploadFiles)
		api.POST("/notebooks/:id/ingest-url", s.IngestURL)
		api.GET("/notebooks/:id/ingest/progress", s.IngestProgress)

		// Chat
		api.POST("/notebooks/:id/chat", s.Chat)
		api.DELETE("/notebooks/:id/chat", s.ClearChatHistory)
		api.POST("/notebooks/:id/messages/truncate", s.TruncateMessages)

		// Global search
		api.POST("/search", s.GlobalSearch)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	s.router = r
}

// Run starts the HTTP server on the configured port.
func (s *Server) Run() error {
	return s.router.Run(":" + s.cfg.ServerPort)
}
