package url

import (
	"context"
	"errors"
	"fmt"
	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/lib/api/random"
	"url-shortener/internal/storage"
)

// AliasLength defines the length of the generated alias.
const AliasLength = 6

// Shorten shortens the given original URL with the provided alias and user email.
// If the alias is empty, a random alias of length AliasLength is generated.
// It returns the alias or an error if the operation fails.
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

	err := s.provider.SaveURL(ctx, alias, originalURL, userEmail)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			return "", domain.ErrAliasExists
		}
		return "", fmt.Errorf("%s: failed to save url: %w", op, err)
	}

	return alias, nil
}
