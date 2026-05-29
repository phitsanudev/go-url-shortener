package application

import "errors"

var (
	ErrInvalidURL = errors.New("invalid url")
	ErrNotFound   = errors.New("short url not found")
	ErrExpired    = errors.New("short url expired")
)
