package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/logger"
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

	start := time.Now()
	resp, err := logger.HTTPClient.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("qdrant ensure_collection failed", "collection", cfg.Collection, "vector_size", vectorSize, "latency_ms", latency, "error", err)
		return fmt.Errorf("connecting to qdrant at %s: %w", cfg.URL, err)
	}
	defer resp.Body.Close()

	// 200 = created, 409 = already exists — both are fine
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		slog.Error("qdrant ensure_collection unexpected status", "collection", cfg.Collection, "status", resp.StatusCode)
		return fmt.Errorf("qdrant returned unexpected status %d when creating collection %q", resp.StatusCode, cfg.Collection)
	}
	slog.Info("qdrant ensure_collection", "collection", cfg.Collection, "vector_size", vectorSize, "status", resp.StatusCode, "latency_ms", latency)
	return nil
}

type CollectionInfo struct {
	Status      string `json:"status"`
	PointsCount int    `json:"points_count"`
	Segments    int    `json:"segments_count"`
	Config      struct {
		Params struct {
			Vectors any `json:"vectors"`
		} `json:"params"`
	} `json:"config"`
	VectorSize int `json:"-"` // Extracted manually
}

type collectionResponse struct {
	Result CollectionInfo `json:"result"`
}

// GetCollectionInfo retrieves metadata and statistics about the configured collection.
func GetCollectionInfo(ctx context.Context, cfg config.QdrantConfig) (*CollectionInfo, error) {
	endpoint := fmt.Sprintf("%s/collections/%s", cfg.URL, cfg.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	if cfg.APIKey != "" {
		req.Header.Set("api-key", cfg.APIKey)
	}

	resp, err := logger.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting collection info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant returned %d when fetching info for %q", resp.StatusCode, cfg.Collection)
	}

	var cr collectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decoding collection info: %w", err)
	}

	info := &cr.Result
	// Try to extract vector size from the any field
	if v, ok := info.Config.Params.Vectors.(map[string]any); ok {
		if size, ok := v["size"].(float64); ok {
			info.VectorSize = int(size)
		} else {
			// It might be a map of named vectors, look for the first one
			for _, val := range v {
				if vm, ok := val.(map[string]any); ok {
					if size, ok := vm["size"].(float64); ok {
						info.VectorSize = int(size)
						break
					}
				}
			}
		}
	}

	return info, nil
}
