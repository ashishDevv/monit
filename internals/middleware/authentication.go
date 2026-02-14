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

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type userCtxKeyType struct{}

var userCtxKey = userCtxKeyType{}

type AuthenticatedUser struct {
	UserID uuid.UUID
	Email  string
}

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
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)

		token, err := a.extractBearerToken(r)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, err.Error())
			return
		}

		claims, err := a.tokenSvc.ValidateAccessToken(token)
		if err != nil {
			utils.FromAppError(w, reqID, err)
			return
		}

		// Extra safety checks (optional but recommended)
		if claims.UserID == "" || claims.Email == "" {
			utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user is unauthorised")
			return
		}

		// Parse UUID here 
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user is unauthorised")
			return
		}
		
		authUser := &AuthenticatedUser{
			UserID: userID,
			Email:  claims.Email,
		}

		newCtx := context.WithValue(ctx, userCtxKey, authUser)
		next.ServeHTTP(w, r.WithContext(newCtx))
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

func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(userCtxKey).(*AuthenticatedUser)
	return user, ok
}
