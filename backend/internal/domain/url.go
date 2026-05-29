package domain

import "time"

type ShortURL struct {
	ID          string
	ShortCode   string
	OriginalURL string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

func (u ShortURL) IsExpired(now time.Time) bool {
	return now.After(u.ExpiresAt)
}
