package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()
	assert.Equal(t, "8083", cfg.Port)
	assert.Equal(t, "mongodb://production-mongo:27017/production?replicaSet=rs0", cfg.MongoURI)
	assert.Equal(t, 500, cfg.DispatchIntervalMS)
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("PRODUCTION_PORT", "9090")
	t.Setenv("PRODUCTION_MONGO_URI", "mongodb://custom/db")
	t.Setenv("PRODUCTION_AMQP_URL", "amqp://custom/")
	t.Setenv("PRODUCTION_DISPATCH_INTERVAL_MS", "1000")

	cfg := Load()
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "mongodb://custom/db", cfg.MongoURI)
	assert.Equal(t, "amqp://custom/", cfg.AMQPURL)
	assert.Equal(t, 1000, cfg.DispatchIntervalMS)
}

func TestLoad_InvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("PRODUCTION_DISPATCH_INTERVAL_MS", "not-a-number")
	cfg := Load()
	assert.Equal(t, 500, cfg.DispatchIntervalMS)
}
