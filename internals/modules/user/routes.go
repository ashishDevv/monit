package user

import "github.com/go-chi/chi/v5"

func Routes(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/{userID}", h.GetUser)
	// r.Post("/{userID}", h.UpdateUserDisplayName)

	return r
}