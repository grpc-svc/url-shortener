package save

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/http-server/middleware/auth"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/metrics"

	"github.com/go-chi/chi/v5/middleware"
)

type Request struct {
	OriginalURL string `json:"original_url" validate:"required"`
	Alias       string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

//go:generate go run github.com/vektra/mockery/v3
type URLShortener interface {
	Shorten(ctx context.Context, originalURL, alias, userEmail string) (string, error)
}

func New(log *slog.Logger, urlShortener URLShortener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.url.save.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("failed to decode request body", slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusBadRequest, resp.Error("invalid request body"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		log.Info("request decoded", slog.Any("req", req))

		ownerEmail, ok := auth.GetEmail(r.Context())
		if !ok {
			log.Error("failed to get owner email from context")

			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to get owner email"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		alias, err := urlShortener.Shorten(r.Context(), req.OriginalURL, req.Alias, ownerEmail)

		if err != nil {
			if errors.Is(err, domain.ErrInvalidURL) || errors.Is(err, domain.ErrInvalidScheme) {
				err = resp.RenderJSON(w, http.StatusBadRequest, resp.Error("invalid URL"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}

				return
			}
			if errors.Is(err, domain.ErrAliasExists) {
				err = resp.RenderJSON(w, http.StatusConflict, resp.Error("alias already exists"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}

			log.Error("failed to shorten url", slog.String("error", err.Error()))
			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("internal error"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		log.Info("url saved successfully", slog.String("alias", alias), slog.String("original_url", req.OriginalURL))

		metrics.URLsCreatedTotal.Inc()

		err = resp.RenderJSON(w, http.StatusOK, Response{
			Response: resp.OK(),
			Alias:    alias,
		})
		if err != nil {
			log.Error("failed to render JSON response", slog.String("error", err.Error()))
		}
	}
}
