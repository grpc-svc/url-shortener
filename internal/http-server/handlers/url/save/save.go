package save

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"url-shortener/internal/http-server/middleware/auth"
	"url-shortener/internal/lib/api/random"
	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/api/urlvalidator"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	OriginalURL string `json:"original_url" validate:"required,url"`
	Alias       string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

const AliasLength = 6

var validate = validator.New()

//go:generate go run github.com/vektra/mockery/v3
type URLSaver interface {
	SaveURL(ctx context.Context, alias, originalURL, ownerEmail string) error
}

func New(log *slog.Logger, urlSaver URLSaver) http.HandlerFunc {
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

		if err = validate.Struct(req); err != nil {
			validationErrs, ok := err.(validator.ValidationErrors)
			if !ok {
				log.Error("failed to validate request", slog.String("error", err.Error()))

				err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to validate request"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}
			log.Error("invalid request", slog.String("error", err.Error()))

			err = resp.RenderJSON(w, http.StatusBadRequest, resp.ValidationError(validationErrs))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		// Validate URL to prevent open redirect vulnerability
		if err = urlvalidator.ValidateURL(req.OriginalURL); err != nil {
			if errors.Is(err, urlvalidator.ErrInvalidURL) {
				log.Error("invalid URL format", slog.String("url", req.OriginalURL), slog.String("error", err.Error()))
			} else if errors.Is(err, urlvalidator.ErrInvalidScheme) {
				log.Warn("blocked non-http(s) URL", slog.String("url", req.OriginalURL), slog.String("error", err.Error()))
			}
			err = resp.RenderJSON(w, http.StatusBadRequest, resp.Error("invalid URL"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		alias := req.Alias
		if alias == "" {
			alias, err = random.NewRandomString(AliasLength)
			if err != nil {
				log.Error("failed to generate random alias", slog.String("error", err.Error()))
				err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to generate alias"))
				if err != nil {
					log.Error("failed to render JSON response", slog.String("error", err.Error()))
				}
				return
			}
		}

		ownerEmail, ok := auth.GetEmail(r.Context())
		if !ok {
			log.Error("failed to get owner email from context")

			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to get owner email"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		err = urlSaver.SaveURL(r.Context(), alias, req.OriginalURL, ownerEmail)
		if errors.Is(err, storage.ErrURLExists) {
			log.Info("alias already exists", slog.String("url", req.OriginalURL))

			err = resp.RenderJSON(w, http.StatusConflict, resp.Error("alias already exists"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		if err != nil {
			log.Error("failed to save url", slog.String("error", err.Error()))

			err = resp.RenderJSON(w, http.StatusInternalServerError, resp.Error("failed to save url"))
			if err != nil {
				log.Error("failed to render JSON response", slog.String("error", err.Error()))
			}
			return
		}

		log.Info("url saved successfully", slog.String("alias", alias), slog.String("original_url", req.OriginalURL))

		err = resp.RenderJSON(w, http.StatusOK, Response{
			Response: resp.OK(),
			Alias:    alias,
		})
		if err != nil {
			log.Error("failed to render JSON response", slog.String("error", err.Error()))
		}
	}
}
