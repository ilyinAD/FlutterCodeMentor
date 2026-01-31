package config

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Database       DatabaseConfig
	Server         ServerConfig
	DeepSeekAPIKey string `env:"DEEPSEEK_API_KEY,required"`
	DeepSeekAPIURL string `env:"DEEPSEEK_API_URL" envDefault:"https://api.deepseek.com/chat/completions"`
}

type DatabaseConfig struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"ilyin-ad"`
	Password string `env:"DB_PASSWORD" envDefault:"postgres"`
	DBName   string `env:"DB_NAME" envDefault:"flutter_code_mentor"`
	SSLMode  string `env:"DB_SSLMODE" envDefault:"disable"`
}

type ServerConfig struct {
	Port string `env:"SERVER_PORT" envDefault:"8080"`
}

func LoadEnv(envPath string) {
	if err := godotenv.Load(envPath); err != nil {
		log.Printf("Warning: .env file not found at %s, using environment variables and defaults", envPath)
	}
}

func Load() *Config {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	return cfg
}

func (c *DatabaseConfig) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.SSLMode,
	)
}
