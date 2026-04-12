package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mfranz/code-gehirn/internal/runtime"
	"github.com/mfranz/code-gehirn/internal/store"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00FFFF")).
			MarginTop(1).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	okStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

	failStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	guidanceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	boldStyle = lipgloss.NewStyle().Bold(true)
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check connectivity to external resources and environment variables",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		fmt.Println(headerStyle.Render("--- 🛠️  Environment Variables ---"))
		printEnvVar("GEHIRN_VAULT_PATH", cfg.VaultPath)
		printEnvVar("GEHIRN_QDRANT_URL", cfg.Qdrant.URL)
		printEnvVar("GEHIRN_QDRANT_COLLECTION", cfg.Qdrant.Collection)
		printEnvVar("GEHIRN_QDRANT_API_KEY", maskKey(cfg.Qdrant.APIKey))
		printEnvVar("GEHIRN_LLM_PROVIDER", cfg.LLM.Provider)
		printEnvVar("GEHIRN_LLM_MODEL", cfg.LLM.Model)
		printEnvVar("GEHIRN_LLM_API_KEY", maskKey(cfg.LLM.APIKey))
		if cfg.LLM.Project != "" {
			printEnvVar("GEHIRN_LLM_PROJECT", cfg.LLM.Project)
		}
		if cfg.LLM.Location != "" {
			printEnvVar("GEHIRN_LLM_LOCATION", cfg.LLM.Location)
		}
		printEnvVar("GEHIRN_EMBEDDING_PROVIDER", cfg.Embedding.Provider)
		printEnvVar("GEHIRN_EMBEDDING_MODEL", cfg.Embedding.Model)
		printEnvVar("GEHIRN_EMBEDDING_API_KEY", maskKey(cfg.Embedding.APIKey))

		// Check for fallback keys
		if os.Getenv("OPENAI_API_KEY") != "" && cfg.LLM.APIKey == "" && cfg.Embedding.APIKey == "" {
			printEnvVar("OPENAI_API_KEY (fallback)", maskKey(os.Getenv("OPENAI_API_KEY")))
		}
		if os.Getenv("ANTHROPIC_API_KEY") != "" && cfg.LLM.APIKey == "" {
			printEnvVar("ANTHROPIC_API_KEY (fallback)", maskKey(os.Getenv("ANTHROPIC_API_KEY")))
		}
		if os.Getenv("GEMINI_API_KEY") != "" && cfg.LLM.APIKey == "" {
			printEnvVar("GEMINI_API_KEY (fallback)", maskKey(os.Getenv("GEMINI_API_KEY")))
		}
		if os.Getenv("GOOGLE_API_KEY") != "" && cfg.LLM.APIKey == "" {
			printEnvVar("GOOGLE_API_KEY (fallback)", maskKey(os.Getenv("GOOGLE_API_KEY")))
		}
		if os.Getenv("GOOGLE_CLOUD_PROJECT") != "" && cfg.LLM.Project == "" {
			printEnvVar("GOOGLE_CLOUD_PROJECT (fallback)", os.Getenv("GOOGLE_CLOUD_PROJECT"))
		}
		if os.Getenv("GOOGLE_CLOUD_LOCATION") != "" && cfg.LLM.Location == "" {
			printEnvVar("GOOGLE_CLOUD_LOCATION (fallback)", os.Getenv("GOOGLE_CLOUD_LOCATION"))
		}
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			printEnvVar("GOOGLE_ADC (fallback)", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		}
		fmt.Println()
		fmt.Println(headerStyle.Render("--- ⚡ Connectivity Checks ---"))

		// Qdrant
		fmt.Printf("%s (%s)... ", boldStyle.Render("Qdrant"), cfg.Qdrant.URL)
		start := time.Now()
		info, err := store.GetCollectionInfo(ctx, cfg.Qdrant)
		duration := time.Since(start).Round(time.Millisecond)
		if err != nil {
			fmt.Printf("%s (%v) - %v\n", failStyle.Render("[✗] FAIL"), duration, err)
			fmt.Println(guidanceStyle.Render(fmt.Sprintf("  Guidance: Ensure Qdrant is running at %s and collection %q exists (run 'code-gehirn index' to create it).", cfg.Qdrant.URL, cfg.Qdrant.Collection)))
		} else {
			fmt.Printf("%s (%v)\n", okStyle.Render("[✓] OK"), duration)
			fmt.Println(infoStyle.Render(fmt.Sprintf("  Collection: %s, points: %d", cfg.Qdrant.Collection, info.PointsCount)))
		}

		// Embedder
		fmt.Printf("%s (%s/%s)... ", boldStyle.Render("Embedder"), cfg.Embedding.Provider, cfg.Embedding.Model)
		start = time.Now()
		embedder, err := runtime.NewEmbedder(*cfg)
		if err != nil {
			fmt.Printf("%s (init) - %v\n", failStyle.Render("[✗] FAIL"), err)
		} else {
			_, err = embedder.EmbedDocuments(ctx, []string{"connectivity check"})
			duration = time.Since(start).Round(time.Millisecond)
			if err != nil {
				fmt.Printf("%s (embed) - %v\n", failStyle.Render("[✗] FAIL"), err)
				fmt.Println(guidanceStyle.Render(fmt.Sprintf("  Guidance: Check GEHIRN_EMBEDDING_API_KEY and network connectivity to %s.", cfg.Embedding.Provider)))
			} else {
				fmt.Printf("%s (%v)\n", okStyle.Render("[✓] OK"), duration)
			}
		}

		// LLM
		fmt.Printf("%s (%s/%s)... ", boldStyle.Render("LLM"), cfg.LLM.Provider, cfg.LLM.Model)
		start = time.Now()
		llm, err := runtime.NewLLM(*cfg)
		if err != nil {
			fmt.Printf("%s (init) - %v\n", failStyle.Render("[✗] FAIL"), err)
		} else {
			_, err = llms.GenerateFromSinglePrompt(ctx, llm, "ping", llms.WithMaxTokens(5))
			duration = time.Since(start).Round(time.Millisecond)
			if err != nil {
				fmt.Printf("%s (generate) - %v\n", failStyle.Render("[✗] FAIL"), err)
				fmt.Println(guidanceStyle.Render(fmt.Sprintf("  Guidance: Check GEHIRN_LLM_API_KEY and network connectivity to %s.", cfg.LLM.Provider)))
			} else {
				fmt.Printf("%s (%v)\n", okStyle.Render("[✓] OK"), duration)
			}
		}

		return nil
	},
}

func printEnvVar(key, value string) {
	fmt.Printf("%s : %s\n", labelStyle.Width(30).Render(key), valueStyle.Render(value))
}

func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
