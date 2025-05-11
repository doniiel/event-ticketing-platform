package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
}

func LoadConfig() *Config {
	grpcPort, err := strconv.Atoi(getEnv("GRPC_PORT", "50051"))
	if err != nil {
		grpcPort = 50051
	}

	httpPort, err := strconv.Atoi(getEnv("HTTP_PORT", "8081"))
	if err != nil {
		httpPort = 8081
	}

	return &Config{
		GRPCPort:    grpcPort,
		HTTPPort:    httpPort,
		DatabaseURL: getEnv("DATABASE_URL", "root:password@tcp(localhost:3306)/events?parseTime=true"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
