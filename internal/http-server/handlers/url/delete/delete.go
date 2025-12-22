package delete

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"url-shortener/internal/http-server/middleware/auth"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:generate go run github.com/vektra/mockery/v3
type URLDeleter interface {
	DeleteURL(ctx context.Context, alias string) error
	GetURLOwner(ctx context.Context, alias string) (string, error)
}

// AdminChecker checks if a user has admin privileges.
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

func New(
	log *slog.Logger,
	urlDeleter URLDeleter,
	adminChecker AdminChecker,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.url.delete.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get user info from context (set by auth middleware)
		userEmail, ok := auth.GetEmail(r.Context())
		if !ok {
			log.Error("failed to get user email from context")
			err := resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to get user email"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}

			return
		}

		userID, ok := auth.GetUID(r.Context())
		if !ok {
			log.Error("failed to get user id from context")
			err := resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to get user id"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("alias parameter is missing")
			err := resp.RenderJSON(w, http.StatusBadRequest, resp.Error("alias parameter is required"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		log = log.With(slog.String("alias", alias), slog.String("user_email", userEmail))

		// Check URL ownership
		ownerEmail, err := urlDeleter.GetURLOwner(r.Context(), alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("alias not found for deletion", slog.String("alias", alias))
			err = resp.RenderJSON(w, http.StatusNotFound, resp.Error("alias not found"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}
		if err != nil {
			log.Error("failed to get URL owner", slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("internal error"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		// Allow deletion if user is owner OR admin
		if ownerEmail != userEmail {
			// Check if user is admin
			isAdmin, err := adminChecker.IsAdmin(r.Context(), userID)
			if err != nil {
				log.Error("failed to check admin status", slog.String("error", err.Error()))
				err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("internal error"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}

			if !isAdmin {
				log.Info("user is not the owner and not an admin",
					slog.String("owner_email", ownerEmail),
				)
				err = resp.RenderJSON(w, http.StatusForbidden, resp.Error("you do not have permission to delete this URL"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}

			log.Info("admin deleting URL", slog.String("owner_email", ownerEmail))
		}

		// Delete URL
		if err = urlDeleter.DeleteURL(r.Context(), alias); err != nil {
			log.Error("failed to delete URL", slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("internal error"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		log.Info("URL deleted successfully")

		err = resp.RenderJSON(w, http.StatusOK, resp.OK())
		if err != nil {
			log.Error("failed to render JSON response", slog.String("error", err.Error()))
		}
	}
}
