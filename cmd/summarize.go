package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/mfranz/code-gehirn/internal/runtime"
	"github.com/mfranz/code-gehirn/internal/summarizer"
	"github.com/spf13/cobra"
)

var summarizeTopK int
var summarizeMaxTokens int

var summarizeCmd = &cobra.Command{
	Use:   "summarize <query>",
	Short: "Summarize search results for a query",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")
		ctx := context.Background()

		// Parallel initialization of store and LLM
		_, qdrantStore, err := runtime.NewEmbedderAndStore(*cfg)
		if err != nil {
			return err
		}

		llm, err := runtime.NewLLM(*cfg)
		if err != nil {
			return err
		}

		if !cmd.Flags().Changed("top") {
			summarizeTopK = cfg.Summary.TopK
		}
		if !cmd.Flags().Changed("tokens") {
			summarizeMaxTokens = cfg.LLM.MaxTokens
		}

		fmt.Printf("Summarizing '%s'...\n", query)
		summary, err := summarizer.Summarize(
			ctx,
			qdrantStore,
			llm,
			query,
			summarizeTopK,
			cfg.VaultPath,
			summarizeMaxTokens,
		)
		if err != nil {
			return err
		}

		fmt.Println("\n---")
		fmt.Println(summary)
		fmt.Println("\n---")
		return nil
	},
}

func init() {
	summarizeCmd.Flags().IntVarP(&summarizeTopK, "top", "k", 5, "number of documents to use for summarization")
	summarizeCmd.Flags().IntVarP(&summarizeMaxTokens, "tokens", "m", 16384, "maximum tokens to generate")
}
