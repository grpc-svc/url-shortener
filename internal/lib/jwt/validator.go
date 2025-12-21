package jwt

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Validator struct {
	publicKey *rsa.PublicKey
}

func New(pemPublicKey string) (*Validator, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pemPublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
	}

	return &Validator{publicKey: key}, nil
}

func (v *Validator) Validate(tokenString string) (*UserClaims, error) {
	var claims UserClaims

	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return v.publicKey, nil
		},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(15*time.Second),
		jwt.WithIssuedAt(),
	)

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	return &claims, nil
}
