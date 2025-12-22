package random

import (
	"crypto/rand"
	"math/big"
)

// NewRandomString generates a cryptographically secure random alphanumeric string of the specified size.
func NewRandomString(size int) string {
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")

	b := make([]rune, size)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			// In case of error, panic as we cannot safely continue without cryptographic randomness
			panic("failed to generate random number: " + err.Error())
		}
		b[i] = chars[num.Int64()]
	}

	return string(b)
}
