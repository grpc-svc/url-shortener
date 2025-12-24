package save_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	domain "url-shortener/internal/domain/url"
	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/http-server/middleware/auth"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name           string
		alias          string
		url            string
		ownerEmail     string
		respError      string
		mockError      error
		mockAlias      string
		statusCode     int
		shouldCallMock bool
	}{
		{
			name:           "Success",
			alias:          "test_alias",
			url:            "https://google.com",
			ownerEmail:     "test@example.com",
			mockAlias:      "test_alias",
			statusCode:     http.StatusOK,
			shouldCallMock: true,
		},
		{
			name:           "Empty alias",
			alias:          "",
			url:            "https://google.com",
			ownerEmail:     "test@example.com",
			mockAlias:      "randomAlias",
			statusCode:     http.StatusOK,
			shouldCallMock: true,
		},
		{
			name:           "Empty URL",
			url:            "",
			alias:          "some_alias",
			ownerEmail:     "test@example.com",
			respError:      "invalid URL",
			mockError:      fmt.Errorf("url.Service.Shorten: %w", domain.ErrInvalidURL),
			statusCode:     http.StatusBadRequest,
			shouldCallMock: true,
		},
		{
			name:           "Shorten Error",
			alias:          "test_alias",
			url:            "https://google.com",
			ownerEmail:     "test@example.com",
			respError:      "internal error",
			mockError:      errors.New("unexpected error"),
			statusCode:     http.StatusInternalServerError,
			shouldCallMock: true,
		},
		{
			name:           "Missing owner email in context",
			alias:          "test_alias",
			url:            "https://google.com",
			ownerEmail:     "", // Empty email means we won't add it to context
			respError:      "failed to get owner email",
			statusCode:     http.StatusInternalServerError,
			shouldCallMock: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewMockURLShortener(t)

			if tc.shouldCallMock {
				urlSaverMock.On("Shorten", mock.Anything, tc.url, tc.alias, tc.ownerEmail).
					Return(tc.mockAlias, tc.mockError).
					Once()
			}

			handler := save.New(slog.New(slog.NewTextHandler(io.Discard, nil)), urlSaverMock)

			input := fmt.Sprintf(`{"original_url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			// Add authenticated user email to context (only if provided)
			if tc.ownerEmail != "" {
				ctx := context.WithValue(req.Context(), auth.ContextKeyEmail, tc.ownerEmail)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.statusCode, rr.Code)

			body := rr.Body.String()

			var resp save.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

		})
	}
}
