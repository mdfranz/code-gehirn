package cmd

import (
	"fmt"
	"log/slog"

	"github.com/mfranz/code-gehirn/internal/runtime"
	"github.com/mfranz/code-gehirn/internal/web"
	"github.com/spf13/cobra"
)

var webPort int

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch the web UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("Initializing providers for web UI...")

		_, qdrantStore, err := runtime.NewEmbedderAndStore(*cfg)
		if err != nil {
			return err
		}

		llm, err := runtime.NewLLM(*cfg)
		if err != nil {
			return err
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
