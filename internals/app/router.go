package app

import (
	middle "project-k/internals/middleware"
	"project-k/internals/modules/user"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func RegisterRoutes(c *Container) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middle.Logger(c.Logger))
	r.Use(middleware.Timeout(5 * time.Second))

	r.Route("/api/v1", func(v1 chi.Router) {
		// v1.Use(c.authMW.Handle)  ->  When you want to apply to all v1 routes

		v1.With(c.authMW.Handle). // when only want to apply to users routes
						Mount("/users", user.Routes(c.userHandler))

		// if you want to apply to some specific routes , then pass it with handler
		//  like this
		// 		v1.Mount("/cart", cart.Routes(c.cartHandler, c.authMW))

		// v1Routes.Mount("/payments", payment)
	})

	return r
}
