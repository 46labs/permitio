package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("port", 7766)
	viper.AutomaticEnv()
}

type Option func(*Config)

func Load(opts ...Option) (*Config, error) {
	// Load schema.yaml
	schemaV := viper.New()
	schemaV.SetConfigName("schema")
	schemaV.SetConfigType("yaml")
	schemaV.AddConfigPath("/config")
	schemaV.AddConfigPath(".")
	_ = schemaV.ReadInConfig()

	// Load data.yaml
	dataV := viper.New()
	dataV.SetConfigName("data")
	dataV.SetConfigType("yaml")
	dataV.AddConfigPath("/config")
	dataV.AddConfigPath(".")
	_ = dataV.ReadInConfig()

	cfg := &Config{
		Port: viper.GetInt("port"),
	}

	if err := schemaV.Unmarshal(&cfg.Schema); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	if err := dataV.Unmarshal(&cfg.Data); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}
