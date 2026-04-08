package config

import (
	"os"
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
}

type SearchConfig struct {
	MinScore float64 `mapstructure:"min_score"`
}

type LogConfig struct {
	File string `mapstructure:"file"`
}

type Config struct {
	Qdrant    QdrantConfig    `mapstructure:"qdrant"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Search    SearchConfig    `mapstructure:"search"`
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
	v.SetDefault("qdrant.collection", "code-gehirn")
	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.model", "gpt-5-mini")
	v.SetDefault("llm.max_tokens", 16384)
	v.SetDefault("embedding.provider", "openai")
	v.SetDefault("embedding.model", "text-embedding-3-small")
	v.SetDefault("search.min_score", 0.0)
	if home, err := os.UserHomeDir(); err == nil {
		v.SetDefault("log.file", home+"/.local/share/code-gehirn/api.log")
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
	if cfg.VaultPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		cfg.VaultPath = wd
	}
	return &cfg, nil
}
