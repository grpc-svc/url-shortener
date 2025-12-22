package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"url-shortener/internal/lib/jwt"

	"github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const (
	ContextKeyUID   contextKey = "uid"
	ContextKeyEmail contextKey = "email"
)

func New(log *slog.Logger, validator *jwt.Validator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		const op = "middleware.auth.New"

		log = log.With(
			slog.String("component", "middleware/auth"),
		)

		log.Info("auth middleware enabled")

		fn := func(w http.ResponseWriter, r *http.Request) {
			log = log.With(
				slog.String("op", op),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			)

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("missing authorization header")
				http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn("invalid authorization header format")
				http.Error(w, "Unauthorized: invalid header format", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]

			claims, err := validator.Validate(tokenStr)
			if err != nil {
				log.Warn("token validation failed", slog.String("error", err.Error()))
				http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
				return
			}

			log.Info("user authenticated",
				slog.Int64("uid", claims.UID),
				slog.String("email", claims.Email),
			)

			ctx := context.WithValue(r.Context(), ContextKeyUID, claims.UID)
			ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}

// GetEmail retrieves the authenticated user's email from the request context.
// Returns the email and true if found, or empty string and false otherwise.
func GetEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(ContextKeyEmail).(string)
	return email, ok
}

// GetUID retrieves the authenticated user's ID from the request context.
// Returns the UID and true if found, or 0 and false otherwise.
func GetUID(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(ContextKeyUID).(int64)
	return uid, ok
}
