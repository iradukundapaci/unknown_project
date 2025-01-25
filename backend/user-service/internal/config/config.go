package config

import (
	"github.com/joho/godotenv"
	"os"
	"strconv"
)

type Config struct {
	Server         ServerConfig
	Services       ServicesConfig
	ClerkSecretKey string
}

type ServerConfig struct {
	Port     int
	GRPCPort int
	Env      string
}

type ServicesConfig struct {
	DBServiceURL      string
	CommentServiceURL string
	StreamServiceURL  string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	grpcPort, _ := strconv.Atoi(os.Getenv("SERVER_GRPC_PORT"))
	port, _ := strconv.Atoi(os.Getenv("SERVER_PORT"))

	return &Config{
		Server: ServerConfig{
			Port:     port,
			GRPCPort: grpcPort,
			Env:      os.Getenv("SERVER_ENV"),
		},
		Services: ServicesConfig{
			DBServiceURL:      os.Getenv("DB_SERVICE_URL"),
			CommentServiceURL: os.Getenv("COMMENT_SERVICE_URL"),
			StreamServiceURL:  os.Getenv("STREAM_SERVICE_URL"),
		},
		ClerkSecretKey: os.Getenv("CLERK_SECRET_KEY"),
	}, nil
}
