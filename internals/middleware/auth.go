package middle

/**
- Work of this file -> Auth package:
	- Validates token
	- Creates claims
	- Stores claims in context
	- Exposes a helper to retrieve claims
**/

import (
	"context"
	"errors"
	"net/http"
	"project-k/internals/security"
	"project-k/pkg/apperror"
	"project-k/pkg/utils"
	"strings"
)

type userCtxKeyType struct{}

var userCtxKey = userCtxKeyType{}

type AuthMiddleware struct {
	tokenSvc *security.TokenService
}

func NewAuthMiddleware(tokenSvc *security.TokenService) *AuthMiddleware {
	return &AuthMiddleware{
		tokenSvc: tokenSvc,
	}
}

func (a *AuthMiddleware) Handle(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		token, err := a.extractBearerToken(r)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "", apperror.Unauthorised, err.Error())
			return
		}

		claims, err := a.tokenSvc.ValidateAccessToken(token)
		if err != nil {
			utils.FromAppError(w, "", err)
			return
		}

		// Extra safety checks (optional but recommended)
		if claims.UserID == "" || claims.Email == "" {
			utils.WriteError(w, http.StatusUnauthorized, "", apperror.Unauthorised, "user is unauhorised")
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	
	return http.HandlerFunc(fn)
}

func (_ *AuthMiddleware) extractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")

	if authHeader == "" {
		return "", errors.New("missing Authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid Authorization header")
	}

	return parts[1], nil
}

func UserFromContext(ctx context.Context) (*security.RequestClaims, bool) {
	claims, ok := ctx.Value(userCtxKey).(*security.RequestClaims)
	return claims, ok
}
