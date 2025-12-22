package delete_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/http-server/handlers/url/delete/mocks"
	"url-shortener/internal/http-server/middleware/auth"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name          string
		alias         string
		ownerEmail    string
		userEmail     string
		userID        int64
		setupMocks    func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker)
		statusCode    int
		withoutEmail  bool
		withoutUserID bool
	}{
		{
			name:       "Success - Owner deletes their URL",
			alias:      "test_alias",
			ownerEmail: "owner@example.com",
			userEmail:  "owner@example.com",
			userID:     123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "test_alias").
					Return("owner@example.com", nil).Once()
				urlDeleter.On("DeleteURL", mock.Anything, "test_alias").
					Return(nil).Once()
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "Success - Admin deletes someone else's URL",
			alias:      "test_alias",
			ownerEmail: "owner@example.com",
			userEmail:  "admin@example.com",
			userID:     456,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "test_alias").
					Return("owner@example.com", nil).Once()
				adminChecker.On("IsAdmin", mock.Anything, int64(456)).
					Return(true, nil).Once()
				urlDeleter.On("DeleteURL", mock.Anything, "test_alias").
					Return(nil).Once()
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "Error - URL not found",
			alias:      "nonexistent",
			ownerEmail: "",
			userEmail:  "user@example.com",
			userID:     123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "nonexistent").
					Return("", storage.ErrURLNotFound).Once()
			},
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Error - User is not owner and not admin",
			alias:      "test_alias",
			ownerEmail: "owner@example.com",
			userEmail:  "other@example.com",
			userID:     789,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "test_alias").
					Return("owner@example.com", nil).Once()
				adminChecker.On("IsAdmin", mock.Anything, int64(789)).
					Return(false, nil).Once()
			},
			statusCode: http.StatusForbidden,
		},
		{
			name:       "Error - GetURLOwner fails with internal error",
			alias:      "test_alias",
			ownerEmail: "",
			userEmail:  "user@example.com",
			userID:     123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "test_alias").
					Return("", errors.New("database error")).Once()
			},
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "Error - AdminChecker fails",
			alias:      "test_alias",
			ownerEmail: "owner@example.com",
			userEmail:  "other@example.com",
			userID:     789,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "test_alias").
					Return("owner@example.com", nil).Once()
				adminChecker.On("IsAdmin", mock.Anything, int64(789)).
					Return(false, errors.New("sso service error")).Once()
			},
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "Error - DeleteURL fails",
			alias:      "test_alias",
			ownerEmail: "owner@example.com",
			userEmail:  "owner@example.com",
			userID:     123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {
				urlDeleter.On("GetURLOwner", mock.Anything, "test_alias").
					Return("owner@example.com", nil).Once()
				urlDeleter.On("DeleteURL", mock.Anything, "test_alias").
					Return(errors.New("database error")).Once()
			},
			statusCode: http.StatusInternalServerError,
		},
		{
			name:         "Error - Missing user email in context",
			alias:        "test_alias",
			ownerEmail:   "owner@example.com",
			userEmail:    "",
			userID:       123,
			withoutEmail: true,
			setupMocks:   func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {},
			statusCode:   http.StatusInternalServerError,
		},
		{
			name:          "Error - Missing user ID in context",
			alias:         "test_alias",
			ownerEmail:    "owner@example.com",
			userEmail:     "user@example.com",
			userID:        0,
			withoutUserID: true,
			setupMocks:    func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {},
			statusCode:    http.StatusInternalServerError,
		},
		{
			name:       "Error - Empty alias parameter",
			alias:      "",
			ownerEmail: "owner@example.com",
			userEmail:  "user@example.com",
			userID:     123,
			setupMocks: func(urlDeleter *mocks.MockURLDeleter, adminChecker *mocks.MockAdminChecker) {},
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlDeleterMock := mocks.NewMockURLDeleter(t)
			adminCheckerMock := mocks.NewMockAdminChecker(t)

			tc.setupMocks(urlDeleterMock, adminCheckerMock)

			handler := delete.New(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				urlDeleterMock,
				adminCheckerMock,
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
