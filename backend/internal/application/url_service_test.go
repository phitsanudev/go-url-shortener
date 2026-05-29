package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bell/go-url-shortener/backend/internal/domain"
)

// Mock repository
type mockURLRepository struct {
	SaveFunc          func(ctx context.Context, item domain.ShortURL) error
	FindByCodeFunc    func(ctx context.Context, code string) (*domain.ShortURL, error)
	DeleteExpiredFunc func(ctx context.Context) error

	SaveCalls          int
	FindByCodeCalls    int
	DeleteExpiredCalls int
}

func (m *mockURLRepository) Save(ctx context.Context, item domain.ShortURL) error {
	m.SaveCalls++
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, item)
	}
	return nil
}

func (m *mockURLRepository) FindByCode(ctx context.Context, code string) (*domain.ShortURL, error) {
	m.FindByCodeCalls++
	if m.FindByCodeFunc != nil {
		return m.FindByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *mockURLRepository) DeleteExpired(ctx context.Context) error {
	m.DeleteExpiredCalls++
	if m.DeleteExpiredFunc != nil {
		return m.DeleteExpiredFunc(ctx)
	}
	return nil
}

// Mock cache
type mockURLCache struct {
	SetFunc    func(ctx context.Context, code string, originalURL string, ttl time.Duration) error
	GetFunc    func(ctx context.Context, code string) (string, error)
	DeleteFunc func(ctx context.Context, code string) error

	SetCalls    int
	GetCalls    int
	DeleteCalls int
}

func (m *mockURLCache) Set(ctx context.Context, code string, originalURL string, ttl time.Duration) error {
	m.SetCalls++
	if m.SetFunc != nil {
		return m.SetFunc(ctx, code, originalURL, ttl)
	}
	return nil
}

func (m *mockURLCache) Get(ctx context.Context, code string) (string, error) {
	m.GetCalls++
	if m.GetFunc != nil {
		return m.GetFunc(ctx, code)
	}
	return "", nil
}

func (m *mockURLCache) Delete(ctx context.Context, code string) error {
	m.DeleteCalls++
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, code)
	}
	return nil
}

func TestURLService_Shorten(t *testing.T) {
	baseURL := "https://short.ly"
	ttl := 24 * time.Hour

	t.Run("Successfully shortens valid URL", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		rawURL := "https://google.com"
		result, err := service.Shorten(context.Background(), rawURL)

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.ShortCode) != 7 {
			t.Errorf("expected shortCode length to be 7, got %d", len(result.ShortCode))
		}

		expectedShortURL := baseURL + "/" + result.ShortCode
		if result.ShortURL != expectedShortURL {
			t.Errorf("expected ShortURL %q, got %q", expectedShortURL, result.ShortURL)
		}

		if repo.SaveCalls != 1 {
			t.Errorf("expected repo.Save to be called 1 time, got %d", repo.SaveCalls)
		}

		if cache.SetCalls != 1 {
			t.Errorf("expected cache.Set to be called 1 time, got %d", cache.SetCalls)
		}
	})

	t.Run("Fails with invalid URL", func(t *testing.T) {
		invalidURLs := []string{
			"google.com",         // Missing scheme
			"ftp://google.com",   // Unsupported scheme
			"http://",            // Missing host
			"plain text",
		}

		for _, rawURL := range invalidURLs {
			t.Run(rawURL, func(t *testing.T) {
				repo := &mockURLRepository{}
				cache := &mockURLCache{}
				service := NewURLService(repo, cache, baseURL, ttl)

				_, err := service.Shorten(context.Background(), rawURL)
				if !errors.Is(err, ErrInvalidURL) {
					t.Errorf("expected error %v, got %v", ErrInvalidURL, err)
				}

				if repo.SaveCalls != 0 {
					t.Errorf("expected repo.Save calls to be 0, got %d", repo.SaveCalls)
				}
			})
		}
	})

	t.Run("Retries on code collision and succeeds", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		// Fail on first 2 attempts, then succeed
		repo.SaveFunc = func(ctx context.Context, item domain.ShortURL) error {
			if repo.SaveCalls <= 2 {
				return errors.New("duplicate key")
			}
			return nil
		}

		rawURL := "https://example.com"
		_, err := service.Shorten(context.Background(), rawURL)

		if err != nil {
			t.Fatalf("expected success after retries, got error: %v", err)
		}

		if repo.SaveCalls != 3 {
			t.Errorf("expected 3 Save attempts, got %d", repo.SaveCalls)
		}

		if cache.SetCalls != 1 {
			t.Errorf("expected cache.Set to be called once, got %d", cache.SetCalls)
		}
	})

	t.Run("Exhausts retries and returns last error on persistent collision", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		dbErr := errors.New("db collision")
		repo.SaveFunc = func(ctx context.Context, item domain.ShortURL) error {
			return dbErr
		}

		rawURL := "https://example.com"
		_, err := service.Shorten(context.Background(), rawURL)

		if !errors.Is(err, dbErr) {
			t.Errorf("expected error %v, got %v", dbErr, err)
		}

		if repo.SaveCalls != 5 {
			t.Errorf("expected exactly 5 retry attempts, got %d", repo.SaveCalls)
		}

		if cache.SetCalls != 0 {
			t.Errorf("expected no cache sets on failure, got %d", cache.SetCalls)
		}
	})
}

func TestURLService_Resolve(t *testing.T) {
	baseURL := "https://short.ly"
	ttl := 24 * time.Hour

	t.Run("Resolves from cache directly (Cache Hit)", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		targetURL := "https://target-url.com"
		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return targetURL, nil
		}

		result, err := service.Resolve(context.Background(), "abc1234")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result != targetURL {
			t.Errorf("expected %q, got %q", targetURL, result)
		}

		if cache.GetCalls != 1 {
			t.Errorf("expected cache.Get to be called 1 time, got %d", cache.GetCalls)
		}

		if repo.FindByCodeCalls != 0 {
			t.Errorf("expected database not to be queried on cache hit, but FindByCode calls = %d", repo.FindByCodeCalls)
		}
	})

	t.Run("Resolves from repo and populates cache (Cache Miss, Repo Hit)", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		targetURL := "https://target-url.com"
		code := "abc1234"
		expiresAt := time.Now().Add(5 * time.Minute)

		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return "", nil // cache miss
		}

		repo.FindByCodeFunc = func(ctx context.Context, c string) (*domain.ShortURL, error) {
			if c == code {
				return &domain.ShortURL{
					ShortCode:   code,
					OriginalURL: targetURL,
					ExpiresAt:   expiresAt,
				}, nil
			}
			return nil, ErrNotFound
		}

		result, err := service.Resolve(context.Background(), code)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result != targetURL {
			t.Errorf("expected %q, got %q", targetURL, result)
		}

		if repo.FindByCodeCalls != 1 {
			t.Errorf("expected repo.FindByCode to be called 1 time, got %d", repo.FindByCodeCalls)
		}

		if cache.SetCalls != 1 {
			t.Errorf("expected cache.Set to be called to populate cache, got %d", cache.SetCalls)
		}
	})

	t.Run("Returns NotFound when code does not exist", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return "", nil // cache miss
		}

		repo.FindByCodeFunc = func(ctx context.Context, code string) (*domain.ShortURL, error) {
			return nil, ErrNotFound
		}

		_, err := service.Resolve(context.Background(), "unknown")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected %v, got %v", ErrNotFound, err)
		}
	})

	t.Run("Returns Expired error and clears cache on expired code", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		code := "oldcode"
		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return "", nil // cache miss
		}

		repo.FindByCodeFunc = func(ctx context.Context, c string) (*domain.ShortURL, error) {
			return &domain.ShortURL{
				ShortCode:   code,
				OriginalURL: "https://expired.com",
				ExpiresAt:   time.Now().Add(-5 * time.Minute), // expired
			}, nil
		}

		_, err := service.Resolve(context.Background(), code)
		if !errors.Is(err, ErrExpired) {
			t.Errorf("expected %v, got %v", ErrExpired, err)
		}

		if cache.DeleteCalls != 1 {
			t.Errorf("expected cache.Delete to be called once on expired check, got %d", cache.DeleteCalls)
		}
	})

	t.Run("Returns error when database query fails with arbitrary error", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, baseURL, ttl)

		dbError := errors.New("connection failed")
		cache.GetFunc = func(ctx context.Context, code string) (string, error) {
			return "", nil
		}
		repo.FindByCodeFunc = func(ctx context.Context, code string) (*domain.ShortURL, error) {
			return nil, dbError
		}

		_, err := service.Resolve(context.Background(), "any")
		if !errors.Is(err, dbError) {
			t.Errorf("expected %v, got %v", dbError, err)
		}
	})
}

func TestURLService_CleanupExpired(t *testing.T) {
	t.Run("Successfully cleans up expired records", func(t *testing.T) {
		repo := &mockURLRepository{}
		cache := &mockURLCache{}
		service := NewURLService(repo, cache, "http://base.ly", time.Hour)

		repo.DeleteExpiredFunc = func(ctx context.Context) error {
			return nil
		}

		err := service.CleanupExpired(context.Background())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if repo.DeleteExpiredCalls != 1 {
			t.Errorf("expected DeleteExpired to be called once, got %d", repo.DeleteExpiredCalls)
		}
	})
}
