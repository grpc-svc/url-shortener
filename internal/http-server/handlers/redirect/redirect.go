package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type URLGetter interface {
	GetURL(ctx context.Context, alias string) (string, error)
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

		originalURL, err := urlGetter.GetURL(r.Context(), alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("alias not found", slog.String("alias", alias))
			err = resp.RenderJSON(w, http.StatusNotFound, resp.Error("alias not found"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}
		if err != nil {
			log.Error("failed to get original URL", slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("internal error"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		// Validate URL to prevent open redirect vulnerability
		parsedURL, err := url.Parse(originalURL)
		if err != nil {
			log.Error("invalid URL format", slog.String("url", originalURL), slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusBadRequest, resp.Error("invalid redirect URL"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		// Only allow http and https schemes to prevent malicious redirects
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			log.Warn("blocked redirect to non-http(s) URL", slog.String("url", originalURL), slog.String("scheme", parsedURL.Scheme))
			err = resp.RenderJSON(w, http.StatusBadRequest, resp.Error("invalid redirect URL"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		http.Redirect(w, r, originalURL, http.StatusFound)
		log.Info("redirected", slog.String("alias", alias), slog.String("original_url", originalURL))
	}
}
