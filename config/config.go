package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DB     DBConfig
	JWT    JWTConfig
	Google GoogleConfig
	REST   ServerConfig
	GRPC   GRPCConfig
	Env    string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type GoogleConfig struct {
	ClientID string
}

type ServerConfig struct {
	Port string
}

type GRPCConfig struct {
	Port string
}

func Load() (*Config, error) {
	cfg := &Config{
		DB: DBConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		JWT: JWTConfig{
			Secret:     os.Getenv("JWT_SECRET"),
			Expiration: 24 * time.Hour,
		},
		Google: GoogleConfig{
			ClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		},
		REST: ServerConfig{
			Port: os.Getenv("REST_PORT"),
		},
		GRPC: GRPCConfig{
			Port: os.Getenv("GRPC_PORT"),
		},
		Env: os.Getenv("APP_ENV"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DB.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.DB.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.DB.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.DB.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Google.ClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}
	if c.REST.Port == "" {
		return fmt.Errorf("REST_PORT is required")
	}
	if c.GRPC.Port == "" {
		return fmt.Errorf("GRPC_PORT is required")
	}
	return nil
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func (c *DBConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}
