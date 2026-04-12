package cmd

import (
	"fmt"
	"os"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/logger"
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
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Once we reach PersistentPreRunE, flags and args have been validated.
		// From this point on, any error returned from RunE is a runtime error,
		// and we don't want to show the usage/help text.
		cmd.SilenceUsage = true

		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return err
		}
		return logger.Init(cfg.Log.AppFile, cfg.Log.APIFile)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.config/code-gehirn/config.yaml or ./config.yaml)")
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(summarizeCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(webCmd)
}
