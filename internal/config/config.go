package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

type QdrantConfig struct {
	URL        string `mapstructure:"url"`
	Collection string `mapstructure:"collection"`
	APIKey     string `mapstructure:"api_key"`
}

type LLMConfig struct {
	Provider  string `mapstructure:"provider"`
	Model     string `mapstructure:"model"`
	APIKey    string `mapstructure:"api_key"`
	BaseURL   string `mapstructure:"base_url"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

type SearchConfig struct {
	MinScore   float64 `mapstructure:"min_score"`
	MaxResults int     `mapstructure:"max_results"`
}

type SummaryConfig struct {
	TopK int `mapstructure:"top_k"`
}

type LogConfig struct {
	APIFile string `mapstructure:"api_file"`
	AppFile string `mapstructure:"app_file"`
}

type Config struct {
	Qdrant    QdrantConfig    `mapstructure:"qdrant"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Search    SearchConfig    `mapstructure:"search"`
	Summary   SummaryConfig   `mapstructure:"summary"`
	Log       LogConfig       `mapstructure:"log"`
	VaultPath string          `mapstructure:"vault_path"`
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(home + "/.config/code-gehirn")
		}
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("GEHIRN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("qdrant.url", "http://localhost:6333")
	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.model", "gpt-5-mini")
	v.SetDefault("llm.max_tokens", 16384)
	v.SetDefault("embedding.provider", "openai")
	v.SetDefault("embedding.model", "text-embedding-3-small")
	v.SetDefault("search.min_score", 0.0)
	v.SetDefault("search.max_results", 15)
	v.SetDefault("summary.top_k", 5)
	if home, err := os.UserHomeDir(); err == nil {
		v.SetDefault("log.api_file", home+"/.local/share/code-gehirn/api.log")
		v.SetDefault("log.app_file", home+"/.local/share/code-gehirn/app.log")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	// Expand ~ in log paths (Viper does not expand tilde in config file or env var values).
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(cfg.Log.APIFile, "~/") {
			cfg.Log.APIFile = home + cfg.Log.APIFile[1:]
		}
		if strings.HasPrefix(cfg.Log.AppFile, "~/") {
			cfg.Log.AppFile = home + cfg.Log.AppFile[1:]
		}
	}
	if cfg.VaultPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		cfg.VaultPath = wd
	}

	// Set default collection name if not provided: code-gehirn-hostname-os-shorthash
	if cfg.Qdrant.Collection == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		// Calculate short hash of the embedding model name
		hash := sha256.Sum256([]byte(cfg.Embedding.Model))
		shortHash := fmt.Sprintf("%x", hash)[:8]
		cfg.Qdrant.Collection = fmt.Sprintf("code-gehirn-%s-%s-%s", hostname, runtime.GOOS, shortHash)
	}

	return &cfg, nil
}
