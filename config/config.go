package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type (
	// Config -.
	Config struct {
		HTTP
		Log
		IdentityServiceAddresses string `env:"IDENTITY_SERVICE_ADDRESSES" env-required:"true"`
		Secret                   string `env-required:"true" env:"ACCESS_TOKEN_SECRET"`
	}

	// HTTP -.
	HTTP struct {
		Port int `env-required:"true" env:"HTTP_PORT" env-default:"8080"`
	}

	// Log -.
	Log struct {
		Level string `env-required:"true" env:"LOG_LEVEL"`
	}
)

// NewConfig returns app config.
func NewConfig() (Config, error) {
	var cfg Config

	err := cleanenv.ReadEnv(&cfg)
	return cfg, err
}
