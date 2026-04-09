package cmd

import (
	"fmt"
	"log/slog"

	"github.com/mfranz/code-gehirn/internal/provider"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/mfranz/code-gehirn/internal/web"
	"github.com/spf13/cobra"
)

var webPort int

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch the web UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("Initializing providers for web UI...")

		embedder, err := provider.NewEmbedder(cfg.Embedding)
		if err != nil {
			return fmt.Errorf("creating embedder: %w", err)
		}

		qdrantStore, err := store.New(cfg.Qdrant, embedder)
		if err != nil {
			return fmt.Errorf("connecting to Qdrant: %w", err)
		}

		llm, err := provider.NewLLM(cfg.LLM)
		if err != nil {
			return fmt.Errorf("connecting to LLM: %w", err)
		}

		server := web.NewServer(*cfg, qdrantStore, llm)
		addr := fmt.Sprintf(":%d", webPort)
		fmt.Printf("Web UI available at http://localhost:%d\n", webPort)
		return server.Start(addr)
	},
}

func init() {
	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Port to listen on")
}
