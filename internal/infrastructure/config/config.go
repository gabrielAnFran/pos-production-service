package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port               string
	MongoURI           string
	AMQPURL            string
	DispatchIntervalMS int
}

func Load() Config {
	return Config{
		Port:               getEnv("PRODUCTION_PORT", "8083"),
		MongoURI:           getEnv("PRODUCTION_MONGO_URI", "mongodb://production-mongo:27017/production?replicaSet=rs0"),
		AMQPURL:            getEnv("PRODUCTION_AMQP_URL", "amqp://guest:guest@localhost:5672/"),
		DispatchIntervalMS: getEnvInt("PRODUCTION_DISPATCH_INTERVAL_MS", 500),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
