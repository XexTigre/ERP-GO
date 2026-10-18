package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL, HTTPAddr string
	MaxConns              int32
}

func Load() (Config, error) {
	c := Config{DatabaseURL: os.Getenv("DATABASE_URL"), HTTPAddr: os.Getenv("HTTP_ADDR"), MaxConns: 12}
	if c.HTTPAddr == "" {
		c.HTTPAddr = "127.0.0.1:8080"
	}
	if raw := os.Getenv("DB_MAX_CONNS"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 1000 {
			return c, fmt.Errorf("DB_MAX_CONNS deve estar entre 1 e 1000")
		}
		c.MaxConns = int32(n)
	}
	return c, nil
}
