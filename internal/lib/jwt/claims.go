package jwt

import "github.com/golang-jwt/jwt/v5"

type UserClaims struct {
	UID   int64  `json:"uid"`
	Email string `json:"email"`
	AppID int    `json:"app_id"`
	jwt.RegisteredClaims
}
