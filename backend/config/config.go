package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

func InitConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Error("No .env file found or error loading .env file")
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Config struct {
	Server Server
	HTTP   HTTP
	APIkey APIkey
}

type Server struct {
	Hostname string `env:"HOSTNAME"`
	Port     string `env:"PORT,notEmpty"`
}

type HTTP struct {
	URL  string `env:"URL"`
	Path string `env:"PATH"`
}

type APIkey struct {
	GeminiAPIkey string `env:"GEMINI_API_KEY"`
	ModelAPIkey  string `env:"MODEL_API_KEY"`
	ModelURL     string `env:"MODEL_URL"`
}
