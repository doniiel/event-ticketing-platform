package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCPort                int
	HTTPPort                int
	DatabaseURL             string
	DatabaseName            string
	EventServiceAddr        string
	NotificationServiceAddr string
}

func LoadConfig() *Config {
	grpcPort, err := strconv.Atoi(getEnv("GRPC_PORT", "50052"))
	if err != nil {
		grpcPort = 50052
	}

	httpPort, err := strconv.Atoi(getEnv("HTTP_PORT", "8082"))
	if err != nil {
		httpPort = 8082
	}

	return &Config{
		GRPCPort:                grpcPort,
		HTTPPort:                httpPort,
		DatabaseURL:             getEnv("DATABASE_URL", "mongodb://localhost:27017"),
		DatabaseName:            getEnv("DATABASE_NAME", "tickets"),
		EventServiceAddr:        getEnv("EVENT_SERVICE_ADDR", "localhost:50051"),
		NotificationServiceAddr: getEnv("NOTIFICATION_SERVICE_ADDR", "localhost:50053"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
