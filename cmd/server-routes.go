package cmd

// Standard library on top, third-party packages below.
import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) routes() {

	// Register static files.
	s.Router.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))
	s.Router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, "public/index.html")
	})

	s.Router.Route("/login", func(r chi.Router) {
		r.Post("/", s.handleAuthLogin())
		r.Post("/reset", s.handleAuthResetPassword())
		r.Post("/update", s.handleAuthUpdatePassword())
	})

	// Unprotected handlers (no authentication required).
	s.Router.Get("/lrs/generate-keys", s.handleVoterGenerateKeys())
	s.Router.Post("/lrs/sign", s.handleVoterSign())
	s.Router.Get("/is-end-of-election", s.handleIsEndOfElection())

	// Admin-only handlers (authentication required).
	s.Router.Route("/admin", func(r chi.Router) {
		r.Get("/users", s.handleAdminGetUsers())
		r.Get("/folded-public-keys", s.handleAdminPutFoldedPublicKeys())
		r.Get("/announce", s.handleAdminAnnounceResult())
	})

	// Voter-only handlers (authentication required).
	s.Router.Route("/voter", func(r chi.Router) {
		r.Patch("/has-voted", s.handleVoterUpdateHasVotedByEmail())
		r.Patch("/keys", s.handleVoterUpdateKeysByEmail())
		r.Post("/private-key", s.handleVoterGetPrivateKeyByEmail())
	})

	// Development-only handlers (no authentication required)
	s.Router.Route("/dev", func(r chi.Router) {
		r.Get("/panic", s.handleDevPanic())
		r.Get("/mem-system", s.handleDevMemSystem())
		r.Get("/mem-app", s.handleDevMemApp())
		r.Get("/db", s.handleDevDatabaseGetFullDatabase())
		r.Get("/db/reset/{schema}/{users}", s.handleDevDatabaseReset())
	})
}
