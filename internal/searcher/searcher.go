package searcher

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// Result wraps a schema.Document with a display title derived from metadata.
type Result struct {
	Doc   schema.Document `json:"doc"`
	Title string          `json:"title"`
	Path  string          `json:"path"`
	Score float32         `json:"score"`
}

// Search performs a similarity search and returns the top-N results above minScore.
func Search(ctx context.Context, store qdrant.Store, query string, topN int, minScore float32) ([]Result, error) {
	start := time.Now()
	docs, err := store.SimilaritySearch(ctx, query, topN,
		vectorstores.WithScoreThreshold(minScore),
	)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("qdrant search failed", "query", query, "top_n", topN, "min_score", minScore, "latency_ms", latency, "error", err)
		return nil, err
	}
	slog.Info("qdrant search", "query", query, "top_n", topN, "min_score", minScore, "results", len(docs), "latency_ms", latency)

	results := make([]Result, len(docs))
	for i, d := range docs {
		results[i] = Result{
			Doc:   d,
			Title: extractTitle(d),
			Path:  extractPath(d),
			Score: d.Score,
		}
	}
	return results, nil
}

func extractPath(d schema.Document) string {
	if v, ok := d.Metadata["source"].(string); ok {
		return v
	}
	return ""
}

func extractTitle(d schema.Document) string {
	if v, ok := d.Metadata["title"].(string); ok && v != "" {
		return v
	}
	if v, ok := d.Metadata["filename"].(string); ok && v != "" {
		return v
	}
	// Fallback: first non-empty line of content
	for _, line := range strings.Split(d.PageContent, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		if line != "" {
			if len(line) > 60 {
				return line[:60] + "..."
			}
			return line
		}
	}
	return "Untitled"
}
