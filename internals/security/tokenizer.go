package security

import (
	"crypto/rsa"
	"os"
	"project-k/config"
	"project-k/pkg/apperror"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	publicKey *rsa.PublicKey
}

func NewTokenService(secretCfg *config.AuthConfig) (*TokenService, error) {
	publicKey, err := loadPublicKey(secretCfg.PublicKeyPath)
	if err != nil {
		// log it here
		return nil, err
	}

	return &TokenService{
		publicKey: publicKey,
	}, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	const op string = "service.token.load_public_key"

	// use if any problem came
	// make path relative to executable if not absolute
	// if !filepath.IsAbs(path) {
	// 	exePath, _ := os.Getwd()  // current working directory
	// 	path = filepath.Join(exePath, path)
	// }

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &apperror.Error{
			Kind:    apperror.Internal,
			Op:      op,
			Message: "error in loading token public key from file path",
			Err:     err,
		}
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return nil, &apperror.Error{
			Kind:    apperror.Internal,
			Op:      op,
			Message: "error in parsing token public key",
			Err:     err,
		}
	}
	return publicKey, nil
}

func (ts *TokenService) ValidateAccessToken(accessToken string) (*RequestClaims, error) {
	const op string = "service.token.validate_access_token"

	claims := &RequestClaims{}

	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return ts.publicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, &apperror.Error{
			Kind:    apperror.Unauthorised,
			Op:      op,
			Message: "invalid access token",
		}
	}

	return claims, nil
}
