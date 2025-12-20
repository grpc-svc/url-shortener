package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name       string
		alias      string
		url        string
		respError  string
		mockError  error
		statusCode int
	}{
		{
			name:       "Success",
			alias:      "test_alias",
			url:        "https://google.com",
			statusCode: http.StatusOK,
		},
		{
			name:       "Empty alias",
			alias:      "",
			url:        "https://google.com",
			statusCode: http.StatusOK,
		},
		{
			name:       "Empty URL",
			url:        "",
			alias:      "some_alias",
			respError:  "field OriginalURL is a required field",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Invalid URL",
			url:        "some invalid URL",
			alias:      "some_alias",
			respError:  "field OriginalURL is not a valid URL",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "SaveURL Error",
			alias:      "test_alias",
			url:        "https://google.com",
			respError:  "failed to save url",
			mockError:  errors.New("unexpected error"),
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewMockURLSaver(t)

			if tc.respError == "" || tc.mockError != nil {
				urlSaverMock.On("SaveURL", mock.Anything, mock.AnythingOfType("string"), tc.url).
					Return(tc.mockError).
					Once()
			}

			handler := save.New(slog.New(slog.NewTextHandler(io.Discard, nil)), urlSaverMock)

			input := fmt.Sprintf(`{"original_url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

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
