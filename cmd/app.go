package cmd

// Standard library on top, third-party packages below.
import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Run is called by main.go and is effectively the entrypoint of the application.
func Run() error {
	var flags = ParseCLI()
	s := Server{}

	// Set up the router.
	s.Router = chi.NewRouter()
	s.middleware()
	s.routes()

	// Set up the database.
	s.URI = &flags.URI
	s.TotalUsers = &flags.TotalUsers
	s.Schema = &flags.Schema
	s.PoolSize = new(int)
	if *s.TotalUsers > 1000 {
		*s.PoolSize = 1000
	} else {
		*s.PoolSize = *s.TotalUsers
	}
	if err := s.database(); err != nil {
		return err
	}

	log.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", s.Router)
}
