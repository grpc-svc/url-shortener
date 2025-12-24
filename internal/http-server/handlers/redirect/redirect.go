package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	domain "url-shortener/internal/domain/url"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type URLGetter interface {
	RedirectURL(ctx context.Context, alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.redirect.New"
		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Error("alias parameter is missing")
			err := resp.RenderJSON(w, http.StatusBadRequest, resp.Error("alias parameter is required"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		originalURL, err := urlGetter.RedirectURL(r.Context(), alias)

		if err != nil {
			if errors.Is(err, domain.ErrURLNotFound) {
				log.Info("alias not found", slog.String("alias", alias))
				err = resp.RenderJSON(w, http.StatusNotFound, resp.Error("not found"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}

			if errors.Is(err, domain.ErrInvalidURL) || errors.Is(err, domain.ErrInvalidScheme) {
				log.Error("invalid url in storage", slog.String("alias", alias), slog.String("error", err.Error()))
				err = resp.RenderJSON(w, http.StatusNotFound, resp.Error("internal error: invalid url stored"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}

			log.Error("failed to get url", slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("internal error"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		http.Redirect(w, r, originalURL, http.StatusFound)
		log.Info("redirected", slog.String("alias", alias), slog.String("original_url", originalURL))

		metrics.RedirectsTotal.Inc()
	}
}
