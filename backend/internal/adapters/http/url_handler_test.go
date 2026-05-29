package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bell/go-url-shortener/backend/internal/application"
	"github.com/bell/go-url-shortener/backend/internal/domain"
)

// Re-define lightweight mocks in HTTP package to keep tests self-contained and fast.
type mockURLRepository struct {
	SaveFunc          func(ctx context.Context, item domain.ShortURL) error
	FindByCodeFunc    func(ctx context.Context, code string) (*domain.ShortURL, error)
	DeleteExpiredFunc func(ctx context.Context) error
}

func (m *mockURLRepository) Save(ctx context.Context, item domain.ShortURL) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, item)
	}
	return nil
}

func (m *mockURLRepository) FindByCode(ctx context.Context, code string) (*domain.ShortURL, error) {
	if m.FindByCodeFunc != nil {
		return m.FindByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *mockURLRepository) DeleteExpired(ctx context.Context) error {
	if m.DeleteExpiredFunc != nil {
		return m.DeleteExpiredFunc(ctx)
	}
	return nil
}

type mockURLCache struct {
	SetFunc    func(ctx context.Context, code string, originalURL string, ttl time.Duration) error
	GetFunc    func(ctx context.Context, code string) (string, error)
	DeleteFunc func(ctx context.Context, code string) error
}

func (m *mockURLCache) Set(ctx context.Context, code string, originalURL string, ttl time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, code, originalURL, ttl)
	}
	return nil
}

func (m *mockURLCache) Get(ctx context.Context, code string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, code)
	}
	return "", nil
}

func (m *mockURLCache) Delete(ctx context.Context, code string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, code)
	}
	return nil
}

func TestURLHandler_HealthCheck(t *testing.T) {
	repo := &mockURLRepository{}
	cache := &mockURLCache{}
	service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
	router := NewRouter(service)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status OK, got %v", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", resp["status"])
	}
}

func TestURLHandler_Shorten(t *testing.T) {
	t.Run("Success 201 Created", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		// Set repo Save to just succeed
		repo.SaveFunc = func(ctx context.Context, item domain.ShortURL) error {
			return nil
		}

		reqBody := `{"url": "https://google.com"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status 201 Created, got %v", rr.Code)
		}

		var resp application.ShortenResult
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.ShortCode) != 7 {
			t.Errorf("expected shortCode length 7, got %d", len(resp.ShortCode))
		}

		if !strings.HasPrefix(resp.ShortURL, "http://localhost:8080/") {
			t.Errorf("expected short url prefix http://localhost:8080/, got %q", resp.ShortURL)
		}
	})

	t.Run("Failure 400 Bad Request - Invalid URL", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		reqBody := `{"url": "invalid-url"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %v", rr.Code)
		}

		var resp errorResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !strings.Contains(resp.Message, "url must start with http:// or https://") {
			t.Errorf("expected error message to contain URL prefix warning, got %q", resp.Message)
		}
	})

	t.Run("Failure 400 Bad Request - Invalid JSON Body", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 Bad Request, got %v", rr.Code)
		}
	})
}

func TestURLHandler_Redirect(t *testing.T) {
	t.Run("Success 302 Redirect", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		targetURL := "https://example.com"
		code := "xyz5678"

		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return targetURL, nil
		}

		req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Errorf("expected status 302 Found, got %v", rr.Code)
		}

		loc := rr.Header().Get("Location")
		if loc != targetURL {
			t.Errorf("expected redirect location %q, got %q", targetURL, loc)
		}
	})

	t.Run("Failure 404 Not Found - Code does not exist", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return "", nil // cache miss
		}

		repo.FindByCodeFunc = func(ctx context.Context, code string) (*domain.ShortURL, error) {
			return nil, application.ErrNotFound
		}

		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404 Not Found, got %v", rr.Code)
		}
	})
}

func TestURLHandler_CleanupExpired(t *testing.T) {
	t.Run("Success 200 OK", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		repo.DeleteExpiredFunc = func(ctx context.Context) error {
			return nil
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cleanup", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200 OK, got %v", rr.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["message"] != "expired urls cleaned" {
			t.Errorf("expected message 'expired urls cleaned', got %q", resp["message"])
		}
	})

	t.Run("Failure 500 Internal Server Error", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := application.NewURLService(repo, cache, "http://localhost:8080", time.Hour)
		router := NewRouter(service)

		repo.DeleteExpiredFunc = func(ctx context.Context) error {
			return errors.New("database breakdown")
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/cleanup", nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 Internal Server Error, got %v", rr.Code)
		}
	})
}
