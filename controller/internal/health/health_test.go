package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthyRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"region":"Mumbai","status":"healthy"}`))
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	result := Check(
		context.Background(),
		client,
		Region{Name: "Mumbai", URL: server.URL},
	)

	if !result.Healthy {
		t.Fatalf("expected healthy region, got: %+v", result)
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", result.StatusCode)
	}
}

func TestUnhealthyService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"region":"Mumbai","status":"unhealthy"}`))
	}))
	defer server.Close()

	result := Check(
		context.Background(),
		&http.Client{Timeout: time.Second},
		Region{Name: "Mumbai", URL: server.URL},
	)

	if result.Healthy {
		t.Fatal("expected unhealthy service to fail health check")
	}
}

func TestWrongRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"region":"Singapore","status":"healthy"}`))
	}))
	defer server.Close()

	result := Check(
		context.Background(),
		&http.Client{Timeout: time.Second},
		Region{Name: "Mumbai", URL: server.URL},
	)

	if result.Healthy {
		t.Fatal("expected region mismatch to fail health check")
	}
}

func TestHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	result := Check(
		context.Background(),
		&http.Client{Timeout: time.Second},
		Region{Name: "Mumbai", URL: server.URL},
	)

	if result.Healthy {
		t.Fatal("expected HTTP 503 to fail health check")
	}

	if result.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected HTTP 503, got %d", result.StatusCode)
	}
}

func TestUnreachableRegion(t *testing.T) {
	result := Check(
		context.Background(),
		&http.Client{Timeout: 100 * time.Millisecond},
		Region{
			Name: "Mumbai",
			URL:  "http://127.0.0.1:1/health",
		},
	)

	if result.Healthy {
		t.Fatal("expected unreachable region to fail health check")
	}

	if result.Error == "" {
		t.Fatal("expected an error for unreachable region")
	}
}
