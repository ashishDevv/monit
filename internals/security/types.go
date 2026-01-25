package security

import "github.com/golang-jwt/jwt/v5"

type RequestClaims struct {
	UserID   string `json:"sub"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	jwt.RegisteredClaims
}