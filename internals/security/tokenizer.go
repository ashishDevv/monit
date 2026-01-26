package security

import (
	"project-k/config"
	"project-k/pkg/apperror"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secret    string
	expiryMin int
}

func NewTokenService(authCfg *config.AuthConfig) *TokenService {
	return &TokenService{
		secret:    authCfg.Secret,
		expiryMin: authCfg.ExpiryMin,
	}
}

func (ts *TokenService) GenerateAccessToken(payload RequestClaims) (string, error) {
	now := time.Now()
	expiryTime := now.Add(time.Duration(ts.expiryMin) * time.Minute)

	payload.ExpiresAt = jwt.NewNumericDate(expiryTime)
	payload.IssuedAt = jwt.NewNumericDate(now)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(ts.secret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (ts *TokenService) ValidateAccessToken(accessToken string) (*RequestClaims, error) {
	const op string = "service.token.validate_access_token"

	claims := &RequestClaims{}

	token, err := jwt.ParseWithClaims(
		accessToken,
		claims,
		func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(ts.secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}),
	)

	if err != nil || !token.Valid {
		return nil, &apperror.Error{
			Kind: apperror.Unauthorised,
			Op: op,
			Message: "invalid token",
		}
	}

	return claims, nil
}
