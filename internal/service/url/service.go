package url

import (
	"context"
	"log/slog"
)

// Provider defines the interface for URL storage operations.
type Provider interface {
	SaveURL(ctx context.Context, alias, originalURL, ownerEmail string) error
	UrlOwner(ctx context.Context, alias string) (string, error)
	DeleteURL(ctx context.Context, alias string) error
	Url(ctx context.Context, alias string) (string, error)
}

type AdminChecker interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type Service struct {
	log          *slog.Logger
	provider     Provider
	adminChecker AdminChecker
}

// New creates a new URL shortening service.
func New(log *slog.Logger, provider Provider, adminChecker AdminChecker) *Service {
	return &Service{
		log:          log,
		provider:     provider,
		adminChecker: adminChecker,
	}
}
