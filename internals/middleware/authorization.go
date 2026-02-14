package middle

import (
	"net/http"
	"project-k/pkg/apperror"
	"project-k/pkg/utils"

	"github.com/go-chi/chi/v5/middleware"
)

func AllowAdmin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		_, ok := UserFromContext(ctx)
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, reqID, apperror.Unauthorised, "user is unauthorised")
			return
		}
 
		// if claims.Role != "admin" {    // will add Role field later
		// 	utils.WriteError(w, http.StatusForbidden, reqID, apperror.Forbidden, "user do not have access")
		// 	return
		// }

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
