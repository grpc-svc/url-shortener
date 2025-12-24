package url

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/storage"
)

func (s *Service) Delete(ctx context.Context, alias, requesterEmail string, requesterID int64) error {
	const op = "url.Service.Delete"

	ownerEmail, err := s.provider.UrlOwner(ctx, alias)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			return fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
		}
		return fmt.Errorf("%s: failed to get url owner: %w", op, err)
	}

	if ownerEmail != requesterEmail {
		isAdmin, err := s.adminChecker.IsAdmin(ctx, requesterID)
		if err != nil {
			return fmt.Errorf("%s: failed to check admin status: %w", op, err)
		}

		if !isAdmin {
			return fmt.Errorf("%s: %w", op, domain.ErrPermissionDenied)
		}

		s.log.Info("admin deleting url", slog.String("alias", alias), slog.String("owner", ownerEmail))
	}

	if err = s.provider.DeleteURL(ctx, alias); err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			return fmt.Errorf("%s: %w", op, domain.ErrURLNotFound)
		}
		return fmt.Errorf("%s: failed to delete url: %w", op, err)
	}

	return nil
}
