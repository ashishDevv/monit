package user

import (
	middle "project-k/internals/middleware"

	"github.com/go-chi/chi/v5"
)

func Routes(h *Handler, authMW *middle.AuthMiddleware) chi.Router {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.LogIn)
	r.With(authMW.Handle).Get("/get-profile", h.GetProfile)

	return r
}


/*
- POST: /users/register  -> register user
	req auth : false
	body : RegisterRequest
	resp : userID

- POST: /users/login   -> login user
	req auth : false
	body : LogInRequest
	resp : LogInResponse

- GET: /users/get-profile -> get user profile
	req auth : true
	body : nil
	resp : GetProfileResponse
*/