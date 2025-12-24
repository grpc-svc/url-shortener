package url

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/lib/api/random"
	"url-shortener/internal/storage"
)

// AliasLength defines the length of the generated alias.
const AliasLength = 6

// URLProvider defines the interface for saving URLs.
type URLProvider interface {
	SaveURL(ctx context.Context, alias, originalURL, ownerEmail string) error
}

type Service struct {
	log         *slog.Logger
	urlProvider URLProvider
}

// New creates a new URL shortening service.
func New(log *slog.Logger, urlProvider URLProvider) *Service {
	return &Service{
		log:         log,
		urlProvider: urlProvider,
	}
}

func (s *Service) Shorten(ctx context.Context, originalURL, alias, userEmail string) (string, error) {
	const op = "url.Service.Shorten"

	if err := domain.ValidateURL(originalURL); err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if alias == "" {
		var err error
		alias, err = random.NewRandomString(AliasLength)
		if err != nil {
			return "", fmt.Errorf("%s: failed to generate alias: %w", op, err)
		}
	}

	err := s.urlProvider.SaveURL(ctx, alias, originalURL, userEmail)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			return "", domain.ErrAliasExists
		}
		return "", fmt.Errorf("%s: failed to save url: %w", op, err)
	}

	return alias, nil
}
