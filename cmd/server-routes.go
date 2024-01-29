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
		//r.Post("/update", s.handleAuthUpdatePassword())
	})

	s.Router.Route("/users", func(r chi.Router) {
		r.Get("/", s.handleGetUsers())
		r.Get("/{email}", s.handleGetUserByEmail())
	})

	s.Router.Route("/keys", func(r chi.Router) {
		r.Get("/public/folded", s.GetFoldedPublicKeysHandler())
		r.Put("/public/folded", s.PutFoldedPublicKeysHandler())
		r.Get("/public/folded/exists", s.ExistsFoldedPublicKeysHandler())
		r.Post("/store", s.StoreUsersPublicKeysHandler())
	})

	s.Router.Route("/fabric/vote", func(r chi.Router) {
		r.Put("/", s.PutVoteHandler())
	})

	// Unprotected handlers (no authentication required).
	s.Router.Get("/lrs/generate-keys", s.handleGenerateKeys())
	s.Router.Post("/lrs/sign", s.handleSign())

	// Admin-only handlers (authentication required).
	s.Router.Route("/admin", func(r chi.Router) {
		r.Get("/users", s.handleAdminGetUsers())
		// r.Get("/email", s.handleAdminSendEmail())
		// r.Get("/announce", s.handleAdminAnnounceResult())
	})

	// Voter-only handlers (authentication required).
	s.Router.Route("/voter", func(r chi.Router) {
		r.Post("/has-voted", s.handleVoterUpdateHasVotedByEmail())
		// r.Post("/private-key", s.handleVoterUpdatePrivateKey())
		// r.Post("/public-key", s.handleVoterUpdatePublicKey())
	})

	// Development-only handlers (no authentication required)
	s.Router.Route("/dev", func(r chi.Router) {
		r.Get("/panic", s.handleDevPanic())
		r.Get("/mem-system", s.handleDevMemSystem())
		r.Get("/mem-app", s.handleDevMemApp())
		r.Get("/db/reset/{schema}/{users}", s.handleDevDatabaseReset())
		r.Get("/db/table", s.handleDevDatabaseGetFullDatabase())
		r.Get("/db/table/users", s.handleDevDatabaseGetUsers())
		r.Get("/db/table/folded-public-keys", s.handleDevDatabaseGetFoldedPublicKeys())
	})
}
