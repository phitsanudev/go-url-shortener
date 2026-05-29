package postgres

import (
	"context"
	"errors"

	"github.com/bell/go-url-shortener/backend/internal/application"
	"github.com/bell/go-url-shortener/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Save(ctx context.Context, item domain.ShortURL) error {
	query := `
		INSERT INTO urls (short_code, original_url, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(ctx, query, item.ShortCode, item.OriginalURL, item.CreatedAt, item.ExpiresAt)
	return err
}

func (r *URLRepository) FindByCode(ctx context.Context, code string) (*domain.ShortURL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, expires_at
		FROM urls
		WHERE short_code = $1
		LIMIT 1
	`

	var item domain.ShortURL

	err := r.db.QueryRow(ctx, query, code).Scan(
		&item.ID,
		&item.ShortCode,
		&item.OriginalURL,
		&item.CreatedAt,
		&item.ExpiresAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, application.ErrNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r *URLRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM urls WHERE expires_at < now()`
	_, err := r.db.Exec(ctx, query)
	return err
}
