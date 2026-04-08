package searcher

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// Result wraps a schema.Document with a display title derived from metadata.
type Result struct {
	Doc   schema.Document
	Title string
	Path  string
	Score float32
}

// Search performs a similarity search and returns the top-N results.
func Search(ctx context.Context, store qdrant.Store, query string, topN int) ([]Result, error) {
	docs, err := store.SimilaritySearch(ctx, query, topN,
		vectorstores.WithScoreThreshold(0.0),
	)
	if err != nil {
		return nil, err
	}

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
