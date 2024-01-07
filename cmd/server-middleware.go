package cmd

// Standard library on top, third-party packages below.
import (
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) middleware() {
	s.Router.Use(middleware.Logger)
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.NoCache)
	s.Router.Use(middleware.Timeout(120 * time.Second))
	s.Router.Use(middleware.StripSlashes)
	s.Router.Use(middleware.Heartbeat("/ping"))
	s.Router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"https://*",
			"http://*",
			"https://voter-app.vercel.app/",
		},
		AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
}
