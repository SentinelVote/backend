package cmd

// Standard library on top, third-party packages below.
import (
	"log"
	"math"
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
	s.URI = flags.URI
	s.TotalUsers = flags.TotalUsers
	s.Schema = flags.Schema
	s.PoolSize = int(math.Ceil(float64(s.TotalUsers) * .75))
	if s.PoolSize > 1000 {
		s.PoolSize = 1000
	} else if s.PoolSize <= 3 {
		s.PoolSize = 4
	}
	if err := s.database(); err != nil {
		return err
	}

	log.Println("Starting server on :8080")
	return http.ListenAndServe(":8080", s.Router)
}
