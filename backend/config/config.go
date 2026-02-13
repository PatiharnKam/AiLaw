package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

func InitConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Error("No .env file found or error loading .env file")
		return nil, err
	}

	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Config struct {
	Server   Server
	Model    Model
	JWT      JWT
	Google   Google
	Database Database `envPrefix:"POSTGRES_"`
	Redis    Redis    `envPrefix:"REDIS_"`
	Quota    Quota    `envPrefix:"QUOTA_"`
	AllowedOrigin []string `env:"ALLOWED_ORIGIN" envSeparator:","`
}

type JWT struct {
	PrivateKey string `env:"JWT_PRIVATE_KEY"`
	PublicKey  string `env:"JWT_PUBLIC_KEY"`
}

type Google struct {
	ClientID     string `env:"GOOGLE_CLIENT_ID"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	RedirectURI  string `env:"GOOGLE_REDIRECT_URI"`
}

type Server struct {
	Hostname string `env:"HOSTNAME"`
	Port     string `env:"PORT,notEmpty"`
}

type Model struct {
	ModelAPIkey       string `env:"MODEL_API_KEY"`
	ModelURL          string `env:"MODEL_URL"`
	ModelCOTURL       string `env:"MODEL_COT_URL"`
	ModelStreamURL    string `env:"MODEL_STREAM_URL"`
	ModelCOTStreamURL string `env:"MODEL_COT_STREAM_URL"`
}

type Database struct {
	PostgresURL               string `env:"URL"`
	PostgresConnMaxLifetime   int    `env:"CONNMAXLIFETIME"`
	PostgresConnMaxIdleTime   int    `env:"CONNMAXIDIETIME"`
	PostgresMaxOpenConns      int    `env:"MAXOPENCONNS"`
	PostgresHealthCheckPeriod int    `env:"HEALTHCHECKPERIOD"`
}

type Redis struct {
	Host     string `env:"HOST"`
	Port     string `env:"PORT"`
	Password string `env:"PASSWORD"`
	DB       int    `env:"DB"`
	URL      string `env:"URL"`
}

type Quota struct {
	DailyLimit      int64 `env:"DAILY_LIMIT"`
	MaxPromptTokens int   `env:"MAX_PROMPT_TOKENS"`
}
