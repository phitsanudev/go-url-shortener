package domain

import (
	"testing"
	"time"
)

func TestShortURL_IsExpired(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		expiresAt time.Time
		now       time.Time
		want      bool
	}{
		{
			name:      "Not Expired - now is before expiration time",
			expiresAt: now.Add(1 * time.Hour),
			now:       now,
			want:      false,
		},
		{
			name:      "Expired - now is after expiration time",
			expiresAt: now.Add(-1 * time.Hour),
			now:       now,
			want:      true,
		},
		{
			name:      "Not Expired - now is exactly at expiration time",
			expiresAt: now,
			now:       now,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := ShortURL{
				ExpiresAt: tt.expiresAt,
			}
			got := u.IsExpired(tt.now)
			if got != tt.want {
				t.Errorf("ShortURL.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
