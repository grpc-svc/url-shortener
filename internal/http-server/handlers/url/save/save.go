package save

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"url-shortener/internal/lib/api/random"
	resp "url-shortener/internal/lib/api/response"
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

type URLSaver interface {
	SaveURL(ctx context.Context, alias, originalURL string) error
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

		if err := validate.Struct(req); err != nil {
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

		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(AliasLength)
		}
		err = urlSaver.SaveURL(r.Context(), alias, req.OriginalURL)
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
