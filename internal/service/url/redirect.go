package url

import (
	"context"
	"errors"
	"fmt"
	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/storage"
)

func (s *Service) RedirectURL(ctx context.Context, alias string) (string, error) {
	const op = "url.Service.GetRedirectURL"

	resURL, err := s.provider.Url(ctx, alias)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			return "", fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: failed to get url: %w", op, err)
	}

	if err = domain.ValidateURL(resURL); err != nil {
		return "", fmt.Errorf("%s: stored url is invalid: %w", op, err)
	}

	return resURL, nil
}
