package config

import (
	"encoding/json"
	"io"
	"os"
)

type Config struct {
	Addr                   string `json:"addr"`
	DBUrl                  string `json:"db_url"`
	AuthTokenExpiryTimeSec int    `json:"auth_token_expiry_sec"`
}

func New(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.DBUrl == "" {
		cfg.DBUrl = "postgres://pam:2932@localhost:15432/pamdb"
	}

	if cfg.Addr == "" {
		cfg.Addr = ":8090"
	}

	if cfg.AuthTokenExpiryTimeSec == 0 {
		cfg.AuthTokenExpiryTimeSec = 300
	}

	return cfg, nil
}
