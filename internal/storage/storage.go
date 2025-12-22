package storage

import (
	"context"
	"errors"
)

var (
	ErrURLNotFound = errors.New("URL not found")
	ErrURLExists   = errors.New("URL already exists")
)

// Storage defines the interface for URL storage operations.
type Storage interface {
	SaveURL(ctx context.Context, alias, originalURL, ownerEmail string) error
	GetURL(ctx context.Context, alias string) (string, error)
	GetURLOwner(ctx context.Context, alias string) (string, error)
	DeleteURL(ctx context.Context, alias string) error
	Close() error
}
