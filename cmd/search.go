package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mfranz/code-gehirn/internal/provider"
	"github.com/mfranz/code-gehirn/internal/searcher"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/spf13/cobra"
)

var searchTopN int
var searchThreshold float64

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Non-interactive semantic search",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		ctx := context.Background()

		embedder, err := provider.NewEmbedder(cfg.Embedding)
		if err != nil {
			return fmt.Errorf("creating embedder: %w", err)
		}
		qdrantStore, err := store.New(cfg.Qdrant, embedder)
		if err != nil {
			return fmt.Errorf("creating store: %w", err)
		}

		results, err := searcher.Search(ctx, qdrantStore, query, searchTopN, float32(searchThreshold))
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		for i, r := range results {
			fmt.Printf("[%d] %s (score: %.3f)\n", i+1, r.Title, r.Score)
			preview := r.Doc.PageContent
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("    %s\n\n", strings.ReplaceAll(preview, "\n", "\n    "))
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchTopN, "top", "n", 5, "number of results to return")
	searchCmd.Flags().Float64VarP(&searchThreshold, "threshold", "t", 0.0, "minimum similarity score (0.0–1.0)")
}
