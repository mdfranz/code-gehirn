package provider

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/mfranz/code-gehirn/internal/config"
	"github.com/mfranz/code-gehirn/internal/logger"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// NewLLM constructs an llms.Model from config.
func NewLLM(cfg config.LLMConfig) (llms.Model, error) {
	switch cfg.Provider {
	case "ollama":
		opts := []ollama.Option{ollama.WithModel(cfg.Model)}
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
		}
		// Ollama doesn't easily take a custom http.Client in langchaingo yet,
		// but it uses the default one which we've already wrapped.
		return ollama.New(opts...)

	case "openai":
		opts := []openai.Option{openai.WithModel(cfg.Model)}
		key := cfg.APIKey
		source := "GEHIRN_LLM_API_KEY"
		if key == "" {
			key = os.Getenv("OPENAI_API_KEY")
			source = "OPENAI_API_KEY (fallback)"
		}
		if key != "" {
			slog.Info("using API key for openai LLM", "source", source)
			opts = append(opts, openai.WithToken(key))
		}
		if cfg.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
		}
		opts = append(opts, openai.WithHTTPClient(logger.HTTPClient))
		return openai.New(opts...)

	case "anthropic":
		opts := []anthropic.Option{anthropic.WithModel(cfg.Model)}
		key := cfg.APIKey
		source := "GEHIRN_LLM_API_KEY"
		if key == "" {
			key = os.Getenv("ANTHROPIC_API_KEY")
			source = "ANTHROPIC_API_KEY (fallback)"
		}
		if key != "" {
			slog.Info("using API key for anthropic LLM", "source", source)
			opts = append(opts, anthropic.WithToken(key))
		}
		opts = append(opts, anthropic.WithHTTPClient(logger.HTTPClient))
		return anthropic.New(opts...)

	case "googleai":
		opts := []googleai.Option{googleai.WithDefaultModel(cfg.Model)}

		// 1. Identify the API key and source
		key := cfg.APIKey
		source := "GEHIRN_LLM_API_KEY"
		if key == "" {
			key = os.Getenv("GEMINI_API_KEY")
			source = "GEMINI_API_KEY (fallback)"
		}
		if key == "" {
			key = os.Getenv("GOOGLE_API_KEY")
			source = "GOOGLE_API_KEY (fallback)"
		}

		// 2. Determine mode and handle environment conflicts
		project := cfg.Project
		if project == "" {
			project = os.Getenv("GOOGLE_CLOUD_PROJECT")
		}

		if key != "" {
			slog.Info("initializing Gemini API (AI Studio) with API key", "source", source)
			// If we have an API key, unset GCP project/location env vars so the SDK
			// does not try to initialize Vertex AI sub-clients via ADC/gRPC.
			os.Unsetenv("GOOGLE_CLOUD_PROJECT")
			os.Unsetenv("GOOGLE_CLOUD_LOCATION")

			// Pass the API key explicitly so it survives for every sub-client the SDK
			// creates internally — including the gRPC-based cache client, which strips
			// the custom HTTP client option (see google/generative-ai-go#151).
			opts = append(opts, googleai.WithAPIKey(key))

			// Also wire up our logging transport for the REST clients.
			httpClient := &http.Client{
				Transport: &googleAuthTransport{
					wrapped: logger.HTTPClient.Transport,
					apiKey:  key,
				},
			}
			opts = append(opts, googleai.WithHTTPClient(httpClient))
			opts = append(opts, googleai.WithRest())
		} else if project != "" {
			slog.Info("initializing Vertex AI (GCP)", "project", project)
			opts = append(opts, googleai.WithCloudProject(project))
			location := cfg.Location
			if location == "" {
				location = os.Getenv("GOOGLE_CLOUD_LOCATION")
			}
			if location != "" {
				opts = append(opts, googleai.WithCloudLocation(location))
			}
			opts = append(opts, googleai.WithHTTPClient(logger.HTTPClient))
			opts = append(opts, googleai.WithRest())
		} else {
			slog.Warn("no API key or project found for googleai, attempting to use Application Default Credentials")
			opts = append(opts, googleai.WithHTTPClient(logger.HTTPClient))
		}

		return googleai.New(context.Background(), opts...)

	default:
		return nil, fmt.Errorf("unknown LLM provider %q: supported providers are ollama, openai, anthropic, googleai", cfg.Provider)
	}
}

// googleAuthTransport injects the API key into the query string of every request.
// This is necessary because the Google SDK ignores WithAPIKey when WithHTTPClient is used.
type googleAuthTransport struct {
	wrapped http.RoundTripper
	apiKey  string
}

func (t *googleAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	q := req.URL.Query()
	if q.Get("key") == "" {
		q.Set("key", t.apiKey)
		req.URL.RawQuery = q.Encode()
	}
	return t.wrapped.RoundTrip(req)
}

// NewEmbedder constructs an embeddings.Embedder from config.
// Anthropic does not expose an embeddings API, so only ollama and openai are supported.
func NewEmbedder(cfg config.EmbeddingConfig) (embeddings.Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		opts := []ollama.Option{ollama.WithModel(cfg.Model)}
		if cfg.BaseURL != "" {
			opts = append(opts, ollama.WithServerURL(cfg.BaseURL))
		}
		llm, err := ollama.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("creating ollama embedder client: %w", err)
		}
		return embeddings.NewEmbedder(llm)

	case "openai":
		opts := []openai.Option{openai.WithEmbeddingModel(cfg.Model)}
		key := cfg.APIKey
		source := "GEHIRN_EMBEDDING_API_KEY"
		if key == "" {
			key = os.Getenv("OPENAI_API_KEY")
			source = "OPENAI_API_KEY (fallback)"
		}
		if key != "" {
			slog.Info("using API key for openai embedder", "source", source)
			opts = append(opts, openai.WithToken(key))
		}
		opts = append(opts, openai.WithHTTPClient(logger.HTTPClient))
		llm, err := openai.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("creating openai embedder client: %w", err)
		}
		return embeddings.NewEmbedder(llm)

	default:
		return nil, fmt.Errorf("unknown embedding provider %q: supported providers are ollama, openai", cfg.Provider)
	}
}
