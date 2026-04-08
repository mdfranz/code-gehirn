package provider

import (
	"context"
	"fmt"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// NewLLM constructs an llms.Model from config.
func NewLLM(cfg config.LLMConfig) (llms.Model, error) {
	switch cfg.Provider {
	case "ollama":
		opts := []ollama.Option{ollama.WithModel(cfg.Model)}
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
		}
		return ollama.New(opts...)

	case "openai":
		opts := []openai.Option{openai.WithModel(cfg.Model)}
		if cfg.APIKey != "" {
			opts = append(opts, openai.WithToken(cfg.APIKey))
		}
		if cfg.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
		}
		return openai.New(opts...)

	case "anthropic":
		opts := []anthropic.Option{anthropic.WithModel(cfg.Model)}
		if cfg.APIKey != "" {
			opts = append(opts, anthropic.WithToken(cfg.APIKey))
		}
		return anthropic.New(opts...)

	case "googleai":
		opts := []googleai.Option{googleai.WithDefaultModel(cfg.Model)}
		if cfg.APIKey != "" {
			opts = append(opts, googleai.WithAPIKey(cfg.APIKey))
		}
		return googleai.New(context.Background(), opts...)

	default:
		return nil, fmt.Errorf("unknown LLM provider %q: supported providers are ollama, openai, anthropic, googleai", cfg.Provider)
	}
}

// NewEmbedder constructs an embeddings.Embedder from config.
// Anthropic does not expose an embeddings API, so only ollama and openai are supported.
func NewEmbedder(cfg config.EmbeddingConfig) (embeddings.Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		opts := []ollama.Option{ollama.WithModel(cfg.Model)}
		llm, err := ollama.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ollama embedder client: %w", err)
		}
		return embeddings.NewEmbedder(llm)

	case "openai":
		opts := []openai.Option{openai.WithEmbeddingModel(cfg.Model)}
		if cfg.APIKey != "" {
			opts = append(opts, openai.WithToken(cfg.APIKey))
		}
		llm, err := openai.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("creating openai embedder client: %w", err)
		}
		return embeddings.NewEmbedder(llm)

	default:
		return nil, fmt.Errorf("unknown embedding provider %q: supported providers are ollama, openai", cfg.Provider)
	}
}
