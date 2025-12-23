package random

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// NewRandomString generates a cryptographically secure random alphanumeric string of the specified size.
func NewRandomString(size int) (string, error) {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")

	b := make([]rune, size)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		b[i] = chars[num.Int64()]
	}

	return string(b), nil
}
