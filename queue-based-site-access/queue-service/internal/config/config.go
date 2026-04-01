package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration populated from environment variables.
type Config struct {
	RedisAddr string // REDIS_ADDR
	TargetURL string // TARGET_URL
	Capacity  int    // QUEUE_CAPACITY — initial max concurrent users
	Port      string // PORT
	WebDir    string // WEB_DIR — path to queue.html and static/ assets
}

func Load() Config {
	capacity := 3
	if v := os.Getenv("QUEUE_CAPACITY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			capacity = n
		}
	}
	return Config{
		RedisAddr: orDefault("REDIS_ADDR", "localhost:6379"),
		TargetURL: orDefault("TARGET_URL", "http://target:80"),
		Capacity:  capacity,
		Port:      orDefault("PORT", "8080"),
		WebDir:    orDefault("WEB_DIR", "./web"),
	}
}

func orDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
