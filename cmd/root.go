package cmd

import (
	"fmt"
	"os"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "code-gehirn",
	Short: "Semantic search and summarization for your markdown knowledge base",
	Long: `code-gehirn indexes a local git repo of markdown files into Qdrant,
enabling semantic search and LLM-powered summarization via an interactive TUI.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.config/code-gehirn/config.yaml or ./config.yaml)")
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(tuiCmd)
}
