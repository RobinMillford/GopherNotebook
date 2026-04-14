package generate

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
        "github.com/yamin/gophernotebook/internal/retrieve"
        "github.com/yamin/gophernotebook/internal/notebook"
)
// Citation represents a specific source reference in the response.
type Citation struct {
	FileName   string `json:"fileName"`
	PageNumber int    `json:"pageNumber"`
	Snippet    string `json:"snippet"`
	Index      int    `json:"index"`
}

// StreamChunk is a piece of the streamed response.
type StreamChunk struct {
	Content   string     `json:"content,omitempty"`
	Citations []Citation `json:"citations,omitempty"`
	Done      bool       `json:"done"`
}

// Generator handles LLM calls using langchaingo.
type Generator struct{}

// NewGenerator creates a new Generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// GenerateStream calls the external LLM with the provided context chunks and
// streams the response via the callback function.
// The LLM provider and API key are passed per-request for privacy.
func (g *Generator) GenerateStream(
	ctx context.Context,
	query string,
	chunks []retrieve.RetrievedChunk,
	provider string,
	apiKey string,
	model string,
        messages []notebook.Message,
	streamFn func(StreamChunk),
) error {
	// Build the LLM client based on provider
	llm, err := g.createLLM(ctx, provider, apiKey, model)
	if err != nil {
		return fmt.Errorf("failed to create LLM: %w", err)
	}

	// Build the prompt with context and citation instructions
	prompt := buildPrompt(query, chunks, messages)

	// Build citations from the context chunks
	citations := buildCitations(chunks)

	// Stream the response
	_, err = llms.GenerateFromSinglePrompt(ctx, llm, prompt,
		llms.WithTemperature(0.3),
		llms.WithMaxTokens(4096),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			streamFn(StreamChunk{
				Content: string(chunk),
			})
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Send citations and done signal
	streamFn(StreamChunk{
		Citations: citations,
		Done:      true,
	})

	return nil
}

// createLLM creates the appropriate langchaingo LLM client.
func (g *Generator) createLLM(ctx context.Context, provider, apiKey, model string) (llms.Model, error) {
	switch strings.ToLower(provider) {
	case "openai", "groq", "openrouter":
		opts := []openai.Option{openai.WithToken(apiKey)}
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		switch strings.ToLower(provider) {
		case "groq":
			opts = append(opts, openai.WithBaseURL("https://api.groq.com/openai/v1"))
		case "openrouter":
			opts = append(opts, openai.WithBaseURL("https://openrouter.ai/api/v1"))
		}
		return openai.New(opts...)

	case "ollama":
		// Ollama exposes an OpenAI-compatible API on localhost:11434.
		// A non-empty token is required by the client library but ignored by Ollama.
		opts := []openai.Option{
			openai.WithToken("ollama"),
			openai.WithBaseURL("http://localhost:11434/v1"),
		}
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		return openai.New(opts...)

	case "lmstudio":
		// LM Studio exposes an OpenAI-compatible API on localhost:1234.
		opts := []openai.Option{
			openai.WithToken("lm-studio"),
			openai.WithBaseURL("http://localhost:1234/v1"),
		}
		if model != "" {
			opts = append(opts, openai.WithModel(model))
		}
		return openai.New(opts...)

	case "google", "gemini":
		return googleai.New(ctx, googleai.WithAPIKey(apiKey), googleai.WithDefaultModel(model))

	case "anthropic":
		opts := []anthropic.Option{anthropic.WithToken(apiKey)}
		if model != "" {
			opts = append(opts, anthropic.WithModel(model))
		}
		return anthropic.New(opts...)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s (supported: openai, google, anthropic, groq, openrouter, ollama, lmstudio)", provider)
	}
}

// buildPrompt constructs the system + user prompt with context chunks.
func buildPrompt(query string, chunks []retrieve.RetrievedChunk, messages []notebook.Message) string {
	var sb strings.Builder

	sb.WriteString("You are a helpful research assistant. Answer the user's question based ONLY on the provided source documents. ")
	sb.WriteString("If the answer cannot be found in the sources, say so clearly. ")
	sb.WriteString("When referencing information, cite the source using [Source N] notation where N is the source number.\n\n")
        if len(messages) > 0 {
                sb.WriteString("=== CHAT HISTORY ===\n")
                for _, m := range messages {
                        sb.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(m.Role), m.Content))
                }
                sb.WriteString("\n")
        }

	sb.WriteString("=== SOURCE DOCUMENTS ===\n\n")
	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("[Source %d] (File: %s, Page: %d)\n", i+1, chunk.FileName, chunk.PageNumber))
		sb.WriteString(chunk.Content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("=== USER QUESTION ===\n")
	sb.WriteString(query)
	sb.WriteString("\n\n")
	sb.WriteString("Answer the question based on the sources above. Cite your sources using [Source N] notation.")

	return sb.String()
}

// buildCitations creates Citation objects from the context chunks.
func buildCitations(chunks []retrieve.RetrievedChunk) []Citation {
	citations := make([]Citation, len(chunks))
	for i, chunk := range chunks {
		// Extract a snippet (first 200 chars)
		snippet := chunk.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		citations[i] = Citation{
			FileName:   chunk.FileName,
			PageNumber: chunk.PageNumber,
			Snippet:    snippet,
			Index:      i + 1,
		}
	}
	return citations
}

