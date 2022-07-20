package healthz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	check := NewHTTP()
	check.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

func TestHealthy(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	check := NewHTTP()
	check.Healthy()
	check.ServeHTTP(rr, req)

	resp := rr.Result()
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSick(t *testing.T) {
	check := NewHTTP()

	// Test that check is initially "healthy".
	{
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		check.Healthy()
		check.ServeHTTP(rr, req)

		resp := rr.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Test that check is now "sick".
	{
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		check.Sick()
		check.ServeHTTP(rr, req)

		resp := rr.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	}
}
