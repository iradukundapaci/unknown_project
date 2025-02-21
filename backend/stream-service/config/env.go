package config

import (
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Load environment variables and handle errors

func LoadEnv() {
	logger := logrus.New()
	err := godotenv.Load()

	if err != nil {
		logger.Warn("Error loading .env file, will use environment variables instead:", err)
		// Don't call Fatal here - continue execution
	}
}
