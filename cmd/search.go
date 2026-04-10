package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mfranz/code-gehirn/internal/runtime"
	"github.com/mfranz/code-gehirn/internal/searcher"
	"github.com/mfranz/code-gehirn/internal/vault"
	"github.com/spf13/cobra"
)

var searchTopN int
var searchThreshold float64
var urlsOnly bool
var allURLs bool

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Non-interactive semantic search",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		ctx := context.Background()

		_, qdrantStore, err := runtime.NewEmbedderAndStore(*cfg)
		if err != nil {
			return err
		}

		if !cmd.Flags().Changed("top") {
			searchTopN = cfg.Search.MaxResults
		}
		if !cmd.Flags().Changed("threshold") {
			searchThreshold = cfg.Search.MinScore
		}

		results, err := searcher.Search(ctx, qdrantStore, query, searchTopN, float32(searchThreshold))
		if err != nil {
			return err
		}

		if len(results) == 0 {
			if !urlsOnly {
				fmt.Println("No results found.")
			}
			return nil
		}

		if urlsOnly {
			re := regexp.MustCompile(`https?://[^\s)\]]+`)
			seen := make(map[string]bool)
			processedFiles := make(map[string]bool)
			for _, r := range results {
				content := r.Doc.PageContent
				if allURLs {
					if processedFiles[r.Path] {
						continue
					}
					processedFiles[r.Path] = true
					fullPath, err := vault.ResolvePath(cfg.VaultPath, r.Path)
					if err != nil {
						continue
					}
					data, err := os.ReadFile(fullPath)
					if err == nil {
						content = string(data)
					}
				}

				matches := re.FindAllString(content, -1)
				for _, m := range matches {
					if !seen[m] {
						fmt.Printf("%s: %s\n", r.Path, m)
						seen[m] = true
					}
				}
			}
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
	searchCmd.Flags().BoolVar(&urlsOnly, "urls", false, "output only extracted URLs")
	searchCmd.Flags().BoolVar(&allURLs, "all", false, "when used with --urls, extract all URLs from the full source file")
}
