package config

import (
	"strings"

	"github.com/spf13/viper"
)

type QdrantConfig struct {
	URL        string `mapstructure:"url"`
	Collection string `mapstructure:"collection"`
	APIKey     string `mapstructure:"api_key"`
}

type LLMConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}

type EmbeddingConfig struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
}

type Config struct {
	Qdrant    QdrantConfig    `mapstructure:"qdrant"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("$HOME/.config/code-gehirn")
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
	v.SetDefault("embedding.provider", "openai")
	v.SetDefault("embedding.model", "text-embedding-3-small")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
