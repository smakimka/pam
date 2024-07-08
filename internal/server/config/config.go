package config

import (
	"os"
	"strconv"
)

type Config struct {
	Addr                   string
	DBUrl                  string
	AuthTokenExpiryTimeSec int
}

func New() (*Config, error) {
	cfg := &Config{
		Addr:  ":8090",
		DBUrl: "postgres://server:server_password@postgres:5432/server_db",
	}

	expiryTime := os.Getenv("AUTH_TOKEN_EXPIRY_TIME_SECONDS")
	expiryTimeSec, err := strconv.Atoi(expiryTime)
	if err != nil {
		return cfg, err
	}

	cfg.AuthTokenExpiryTimeSec = expiryTimeSec

	return cfg, nil
}
