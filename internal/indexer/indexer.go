package indexer

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

// Run walks repoPath, finds all .md files, chunks them with heading-aware
// splitting, and upserts the chunks into Qdrant via store.AddDocuments.
// progressFn is called after each file (may be nil).
func Run(ctx context.Context, repoPath string, store qdrant.Store, progressFn func(file string, chunks int)) error {
	splitter := textsplitter.NewMarkdownTextSplitter(
		textsplitter.WithChunkSize(500),
		textsplitter.WithChunkOverlap(50),
	)

	return filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}

		docs, err := loadAndChunk(ctx, path, repoPath, splitter)
		if err != nil {
			return err
		}
		if len(docs) == 0 {
			return nil
		}

		rel, _ := filepath.Rel(repoPath, path)
		start := time.Now()
		if _, err := store.AddDocuments(ctx, docs); err != nil {
			slog.Error("qdrant upsert failed", "file", rel, "chunks", len(docs), "error", err)
			return err
		}
		slog.Info("qdrant upsert", "file", rel, "chunks", len(docs), "latency_ms", time.Since(start).Milliseconds())
		if progressFn != nil {
			progressFn(rel, len(docs))
		}
		return nil
	})
}

func loadAndChunk(ctx context.Context, path, repoPath string, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	loader := documentloaders.NewText(f)
	docs, err := loader.LoadAndSplit(ctx, splitter)
	if err != nil {
		return nil, err
	}

	rel, _ := filepath.Rel(repoPath, path)
	base := filepath.Base(path)
	// Strip .md extension for a cleaner display title
	title := strings.TrimSuffix(base, filepath.Ext(base))

	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}
		docs[i].Metadata["source"] = rel
		docs[i].Metadata["filename"] = base
		docs[i].Metadata["title"] = title
	}
	return docs, nil
}
