package runtime

import (
	"fmt"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/provider"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// NewEmbedder initializes the configured embedding provider.
func NewEmbedder(cfg config.Config) (embeddings.Embedder, error) {
	embedder, err := provider.NewEmbedder(cfg.Embedding)
	if err != nil {
		return nil, fmt.Errorf("creating embedder: %w", err)
	}
	return embedder, nil
}

// NewStore initializes the configured vector store with the given embedder.
func NewStore(cfg config.Config, embedder embeddings.Embedder) (qdrant.Store, error) {
	qdrantStore, err := store.New(cfg.Qdrant, embedder)
	if err != nil {
		return qdrant.Store{}, fmt.Errorf("creating store: %w", err)
	}
	return qdrantStore, nil
}

// NewLLM initializes the configured LLM provider.
func NewLLM(cfg config.Config) (llms.Model, error) {
	llm, err := provider.NewLLM(cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("connecting to LLM: %w", err)
	}
	return llm, nil
}

// NewEmbedderAndStore initializes embedder first, then store.
func NewEmbedderAndStore(cfg config.Config) (embeddings.Embedder, qdrant.Store, error) {
	embedder, err := NewEmbedder(cfg)
	if err != nil {
		return nil, qdrant.Store{}, err
	}
	qdrantStore, err := NewStore(cfg, embedder)
	if err != nil {
		return nil, qdrant.Store{}, err
	}
	return embedder, qdrantStore, nil
}
