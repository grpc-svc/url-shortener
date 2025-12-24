package url

import (
	"errors"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		{
			name:    "Valid HTTP URL",
			url:     "http://example.com",
			wantErr: nil,
		},
		{
			name:    "Valid HTTPS URL",
			url:     "https://example.com",
			wantErr: nil,
		},
		{
			name:    "Valid HTTPS URL with path",
			url:     "https://example.com/path/to/page",
			wantErr: nil,
		},
		{
			name:    "Valid HTTPS URL with query params",
			url:     "https://example.com/path?key=value&foo=bar",
			wantErr: nil,
		},
		{
			name:    "Invalid URL format",
			url:     "not a url",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "FTP scheme (not allowed)",
			url:     "ftp://example.com",
			wantErr: ErrInvalidScheme,
		},
		{
			name:    "File scheme (not allowed)",
			url:     "file:///etc/passwd",
			wantErr: ErrInvalidScheme,
		},
		{
			name:    "Javascript scheme (not allowed)",
			url:     "javascript:alert('xss')",
			wantErr: ErrInvalidScheme,
		},
		{
			name:    "Data scheme (not allowed)",
			url:     "data:text/html,<script>alert('xss')</script>",
			wantErr: ErrInvalidScheme,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "Missing scheme",
			url:     "example.com",
			wantErr: ErrInvalidURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateURL(tt.url)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateURL() expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}
