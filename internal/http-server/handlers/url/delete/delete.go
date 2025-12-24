package delete

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/http-server/middleware/auth"
	resp "url-shortener/internal/lib/api/response"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:generate go run github.com/vektra/mockery/v3
type URLDeleter interface {
	Delete(ctx context.Context, alias, requesterEmail string, requesterID int64) error
}

func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
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

		err := urlDeleter.Delete(r.Context(), alias, userEmail, userID)

		if err != nil {
			if errors.Is(err, domain.ErrURLNotFound) {
				log.Info("url not found", slog.String("alias", alias))
				err = resp.RenderJSON(w, http.StatusNotFound, resp.Error("not found"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}
			if errors.Is(err, domain.ErrPermissionDenied) {
				log.Info("permission denied", slog.String("alias", alias), slog.String("user", userEmail))
				err = resp.RenderJSON(w, http.StatusForbidden, resp.Error("permission denied"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}

			log.Error("failed to delete url", slog.String("error", err.Error()))
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
