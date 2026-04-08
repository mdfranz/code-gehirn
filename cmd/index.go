package cmd

import (
	"context"
	"fmt"

	"github.com/mfranz/code-gehirn/internal/indexer"
	"github.com/mfranz/code-gehirn/internal/provider"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index <path>",
	Short: "Index markdown files from a git repo into Qdrant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := args[0]
		ctx := context.Background()

		embedder, err := provider.NewEmbedder(cfg.Embedding)
		if err != nil {
			return fmt.Errorf("creating embedder: %w", err)
		}

		qdrantStore, err := store.New(cfg.Qdrant, embedder)
		if err != nil {
			return fmt.Errorf("creating store: %w", err)
		}

		// Probe embedding dimension
		vec, err := embedder.EmbedQuery(ctx, "probe")
		if err != nil {
			return fmt.Errorf("probing embedding dimension: %w", err)
		}
		if err := store.EnsureCollection(ctx, cfg.Qdrant, len(vec)); err != nil {
			return fmt.Errorf("ensuring collection: %w", err)
		}

		fmt.Printf("Indexing %s into collection %q...\n", repoPath, cfg.Qdrant.Collection)
		total := 0
		err = indexer.Run(ctx, repoPath, qdrantStore, func(file string, chunks int) {
			total += chunks
			fmt.Printf("  %s (%d chunks)\n", file, chunks)
		})
		if err != nil {
			return err
		}
		fmt.Printf("\nDone. %d chunks indexed.\n", total)
		return nil
	},
}
