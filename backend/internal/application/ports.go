package application

import (
	"context"
	"time"

	"github.com/bell/go-url-shortener/backend/internal/domain"
)

type URLRepository interface {
	Save(ctx context.Context, item domain.ShortURL) error
	FindByCode(ctx context.Context, code string) (*domain.ShortURL, error)
	DeleteExpired(ctx context.Context) error
}

type URLCache interface {
	Set(ctx context.Context, code string, originalURL string, ttl time.Duration) error
	Get(ctx context.Context, code string) (string, error)
	Delete(ctx context.Context, code string) error
}
