package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCPort          int
	HTTPPort          int
	DatabaseURL       string
	TicketServiceAddr string
}

func LoadConfig() *Config {
	grpcPort, err := strconv.Atoi(getEnv("GRPC_PORT", "50053"))
	if err != nil {
		grpcPort = 50053
	}

	httpPort, err := strconv.Atoi(getEnv("HTTP_PORT", "8083"))
	if err != nil {
		httpPort = 8083
	}

	return &Config{
		GRPCPort:          grpcPort,
		HTTPPort:          httpPort,
		DatabaseURL:       getEnv("DATABASE_URL", "root:password@tcp(localhost:3306)/notifications?parseTime=true"),
		TicketServiceAddr: getEnv("TICKET_SERVICE_ADDR", "localhost:50052"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
