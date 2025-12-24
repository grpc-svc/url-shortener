package delete_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/domain/url"
	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/delete/mocks"
	"url-shortener/internal/http-server/middleware/auth"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name          string
		alias         string
		userEmail     string
		userID        int64
		setupMocks    func(urlDeleter *mocks.MockURLDeleter)
		statusCode    int
		withoutEmail  bool
		withoutUserID bool
	}{
		{
			name:      "Success - Owner deletes their URL",
			alias:     "test_alias",
			userEmail: "owner@example.com",
			userID:    123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter) {
				urlDeleter.On("Delete", mock.Anything, "test_alias", "owner@example.com", int64(123)).
					Return(nil).Once()
			},
			statusCode: http.StatusOK,
		},
		{
			name:      "Success - Admin deletes someone else's URL",
			alias:     "test_alias",
			userEmail: "admin@example.com",
			userID:    456,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter) {
				urlDeleter.On("Delete", mock.Anything, "test_alias", "admin@example.com", int64(456)).
					Return(nil).Once()
			},
			statusCode: http.StatusOK,
		},
		{
			name:      "Error - URL not found",
			alias:     "nonexistent",
			userEmail: "user@example.com",
			userID:    123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter) {
				urlDeleter.On("Delete", mock.Anything, "nonexistent", "user@example.com", int64(123)).
					Return(url.ErrURLNotFound).Once()
			},
			statusCode: http.StatusNotFound,
		},
		{
			name:      "Error - User is not owner and not admin",
			alias:     "test_alias",
			userEmail: "other@example.com",
			userID:    789,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter) {
				urlDeleter.On("Delete", mock.Anything, "test_alias", "other@example.com", int64(789)).
					Return(url.ErrPermissionDenied).Once()
			},
			statusCode: http.StatusForbidden,
		},
		{
			name:      "Error - Delete fails with internal error",
			alias:     "test_alias",
			userEmail: "user@example.com",
			userID:    123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter) {
				urlDeleter.On("Delete", mock.Anything, "test_alias", "user@example.com", int64(123)).
					Return(errors.New("database error")).Once()
			},
			statusCode: http.StatusInternalServerError,
		},
		{
			name:         "Error - Missing user email in context",
			alias:        "test_alias",
			userEmail:    "",
			userID:       123,
			withoutEmail: true,
			setupMocks:   func(urlDeleter *mocks.MockURLDeleter) {},
			statusCode:   http.StatusInternalServerError,
		},
		{
			name:          "Error - Missing user ID in context",
			alias:         "test_alias",
			userEmail:     "user@example.com",
			userID:        0,
			withoutUserID: true,
			setupMocks:    func(urlDeleter *mocks.MockURLDeleter) {},
			statusCode:    http.StatusInternalServerError,
		},
		{
			name:       "Error - Empty alias parameter",
			alias:      "",
			userEmail:  "user@example.com",
			userID:     123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter) {},
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// t.Parallel()

			urlDeleterMock := mocks.NewMockURLDeleter(t)

			tc.setupMocks(urlDeleterMock)

			handler := delete.New(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				urlDeleterMock,
			)

			req, err := http.NewRequest(http.MethodDelete, "/"+tc.alias, nil)
			require.NoError(t, err)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			// Add authenticated user email to context (unless we're testing missing email)
			if !tc.withoutEmail {
				ctx := context.WithValue(req.Context(), auth.ContextKeyEmail, tc.userEmail)
				req = req.WithContext(ctx)
			}

			// Add authenticated user ID to context (unless we're testing missing ID)
			if !tc.withoutUserID {
				ctx := context.WithValue(req.Context(), auth.ContextKeyUID, tc.userID)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.statusCode, rr.Code)
		})
	}
}
