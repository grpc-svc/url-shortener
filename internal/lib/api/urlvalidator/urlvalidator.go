package urlvalidator

import (
	"errors"
	"net/url"
)

var (
	// ErrInvalidURL indicates that the URL format is invalid
	ErrInvalidURL = errors.New("invalid URL format")
	// ErrInvalidScheme indicates that the URL scheme is not allowed
	ErrInvalidScheme = errors.New("only http and https schemes are allowed")
)

// ValidateURL validates that the URL has correct format and uses http/https scheme
// to prevent open redirect vulnerabilities and malicious redirects
func ValidateURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ErrInvalidURL
	}

	if parsedURL.Scheme == "" {
		return ErrInvalidURL
	}

	// Only allow http and https schemes
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return ErrInvalidScheme
	}

	if parsedURL.Host == "" {
		return ErrInvalidURL
	}

	return nil
}
