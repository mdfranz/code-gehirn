package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// New creates a qdrant.Store configured from cfg using the given embedder.
func New(cfg config.QdrantConfig, embedder embeddings.Embedder) (qdrant.Store, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return qdrant.Store{}, fmt.Errorf("invalid qdrant URL %q: %w", cfg.URL, err)
	}
	opts := []qdrant.Option{
		qdrant.WithURL(*u),
		qdrant.WithCollectionName(cfg.Collection),
		qdrant.WithEmbedder(embedder),
	}
	if cfg.APIKey != "" {
		opts = append(opts, qdrant.WithAPIKey(cfg.APIKey))
	}
	return qdrant.New(opts...)
}

// EnsureCollection creates the Qdrant collection if it doesn't already exist.
// vectorSize must match the embedding model's output dimension.
func EnsureCollection(ctx context.Context, cfg config.QdrantConfig, vectorSize int) error {
	body, err := json.Marshal(map[string]any{
		"vectors": map[string]any{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/collections/%s", cfg.URL, cfg.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("api-key", cfg.APIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to qdrant at %s: %w", cfg.URL, err)
	}
	defer resp.Body.Close()

	// 200 = created, 409 = already exists — both are fine
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("qdrant returned unexpected status %d when creating collection %q", resp.StatusCode, cfg.Collection)
	}
	return nil
}
