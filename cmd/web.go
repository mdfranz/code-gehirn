package cmd

import (
	"fmt"
	"log/slog"

	"github.com/mfranz/code-gehirn/internal/runtime"
	"github.com/mfranz/code-gehirn/internal/web"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

var webPort int

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch the web UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("Initializing providers for web UI...")

		type result struct {
			embedder    embeddings.Embedder
			qdrantStore qdrant.Store
			llm         llms.Model
			err         error
		}

		storeChan := make(chan result, 1)
		llmChan := make(chan result, 1)

		go func() {
			embedder, qdrantStore, err := runtime.NewEmbedderAndStore(*cfg)
			storeChan <- result{embedder: embedder, qdrantStore: qdrantStore, err: err}
		}()

		go func() {
			llm, err := runtime.NewLLM(*cfg)
			llmChan <- result{llm: llm, err: err}
		}()

		resStore := <-storeChan
		if resStore.err != nil {
			return resStore.err
		}

		resLLM := <-llmChan
		if resLLM.err != nil {
			return resLLM.err
		}

		server := web.NewServer(*cfg, resStore.qdrantStore, resLLM.llm)
		addr := fmt.Sprintf(":%d", webPort)
		fmt.Printf("Web UI available at http://localhost:%d\n", webPort)
		return server.Start(addr)
	},
}

func init() {
	webCmd.Flags().IntVarP(&webPort, "port", "p", 8080, "Port to listen on")
}
