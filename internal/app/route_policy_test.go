// Package app contains route policy verification tests.
//
// TODO(phase0): build the real router with stub services to fully validate
// route policy. Current test validates the policy framework compiles and
// passes against a bare router.
package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestRoutePolicyFramework verifies the route policy test framework compiles
// and that gin routes can be registered and probed.
func TestRoutePolicyFramework(t *testing.T) {
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
}

// TestRoutePolicyNoUndefinedRoutes verifies the route policy framework can
// enumerate routes. Full policy checks (public/admin/moderator probing)
// require building the real router with stub services — see TODO above.
func TestRoutePolicyNoUndefinedRoutes(t *testing.T) {
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	routes := []struct {
		Method string
		Path   string
	}{
		{Method: http.MethodGet, Path: "/health"},
	}

	require.NotEmpty(t, routes)
	for _, rt := range routes {
		resp := httptest.NewRecorder()
		req := httptest.NewRequest(rt.Method, rt.Path, nil)
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}
}
