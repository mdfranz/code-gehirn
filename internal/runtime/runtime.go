package runtime

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/provider"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// NewEmbedder initializes the configured embedding provider.
func NewEmbedder(cfg config.Config) (embeddings.Embedder, error) {
	slog.Info("Initializing embedder",
		"provider", cfg.Embedding.Provider,
		"model", cfg.Embedding.Model)
	embedder, err := provider.NewEmbedder(cfg.Embedding)
	if err != nil {
		return nil, fmt.Errorf("creating embedder: %w", err)
	}
	return embedder, nil
}

// NewStore initializes the configured vector store with the given embedder.
func NewStore(cfg config.Config, embedder embeddings.Embedder) (qdrant.Store, error) {
	slog.Info("Initializing vector store",
		"url", cfg.Qdrant.URL,
		"collection", cfg.Qdrant.Collection)
	qdrantStore, err := store.New(cfg.Qdrant, embedder)
	if err != nil {
		return qdrant.Store{}, fmt.Errorf("creating store: %w", err)
	}
	return qdrantStore, nil
}

// GetCollectionInfo retrieves and logs detailed collection info.
func GetCollectionInfo(cfg config.Config) (*store.CollectionInfo, error) {
	info, err := store.GetCollectionInfo(context.Background(), cfg.Qdrant)
	if err != nil {
		slog.Error("Failed to fetch collection metadata", "collection", cfg.Qdrant.Collection, "error", err)
		return nil, err
	}
	slog.Info("Collection metadata",
		"collection", cfg.Qdrant.Collection,
		"status", info.Status,
		"points", info.PointsCount,
		"vector_size", info.VectorSize,
		"segments", info.Segments)
	return info, nil
}

// NewLLM initializes the configured LLM provider.
func NewLLM(cfg config.Config) (llms.Model, error) {
	slog.Info("Initializing LLM",
		"provider", cfg.LLM.Provider,
		"model", cfg.LLM.Model)
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
