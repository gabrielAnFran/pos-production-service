package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHealthHandler(nil)
	r := gin.New()
	r.GET("/healthz", h.Healthz)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestReadyz_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// A client pointed at a non-routable address never completes its
	// handshake, so Ping reliably fails within the handler's timeout.
	opts := options.Client().ApplyURI("mongodb://10.255.255.1:27017").SetServerSelectionTimeout(1 * time.Second)
	client, err := mongo.Connect(t.Context(), opts)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(t.Context()) })

	h := NewHealthHandler(client)
	r := gin.New()
	r.GET("/readyz", h.Readyz)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
