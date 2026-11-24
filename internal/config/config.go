package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration

	DBMaxConns        int32
	DBMinConns        int32
	DBConnectTimeout  time.Duration
	DBQueryTimeout    time.Duration
	DBMaxConnLifetime time.Duration

	PositionCheckInterval time.Duration
	PositionStaleAfter    time.Duration
	PositionWorkers       int
	PositionQueueSize     int
	PushHTTPURL           string
	PushRequestTimeout    time.Duration
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:          getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		LogLevel:          getenv("LOG_LEVEL", "info"),
		ShutdownTimeout:   getdur("SHUTDOWN_TIMEOUT", 10*time.Second),
		DBMaxConns:        int32(getint("DATABASE_MAX_CONNS", 10)),
		DBMinConns:        int32(getint("DATABASE_MIN_CONNS", 1)),
		DBConnectTimeout:  getdur("DATABASE_CONNECT_TIMEOUT", 5*time.Second),
		DBQueryTimeout:    getdur("DATABASE_QUERY_TIMEOUT", 3*time.Second),
		DBMaxConnLifetime: getdur("DATABASE_MAX_CONN_LIFETIME", time.Hour),
		PositionCheckInterval: getdur("POSITION_CHECK_INTERVAL", 30*time.Second),
		PositionStaleAfter:    getdur("POSITION_STALE_AFTER", 60*time.Second),
		PositionWorkers:       getint("POSITION_WORKERS", 4),
		PositionQueueSize:     getint("POSITION_QUEUE_SIZE", 100),
		PushHTTPURL:           os.Getenv("PUSH_HTTP_URL"),
		PushRequestTimeout:    getdur("PUSH_REQUEST_TIMEOUT", time.Second),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getint(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}