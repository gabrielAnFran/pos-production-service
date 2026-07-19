package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorrelationID_GeneratesWhenAbsent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CorrelationID())
	var seen string
	r.GET("/x", func(c *gin.Context) {
		seen = GetCorrelationID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	require.NotEmpty(t, seen)
	assert.Equal(t, seen, w.Header().Get(CorrelationIDHeader))
}

func TestCorrelationID_PropagatesExisting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CorrelationID())
	var seen string
	r.GET("/x", func(c *gin.Context) {
		seen = GetCorrelationID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(CorrelationIDHeader, "fixed-id")
	r.ServeHTTP(w, req)

	assert.Equal(t, "fixed-id", seen)
	assert.Equal(t, "fixed-id", w.Header().Get(CorrelationIDHeader))
}
