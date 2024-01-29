package cmd

import (
	"backend/internal/db"
	"context"
	"github.com/goccy/go-json"
	"log"
	"net/http"
)

// +----------------------------------------------------------------------------------------------+
// |                                        Voter Handlers                                        |
// +----------------------------------------------------------------------------------------------+

func (s *Server) handleVoterUpdateHasVotedByEmail() http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		HasVoted bool   `json:"hasVoted"`
	}
	type response struct {
		Success bool `json:"success"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Called: handleVoterUpdateHasVotedByEmail")
		if !isHeaderJSON(w, r) {
			return
		}
		defer r.Body.Close()

		// Define a request structure to match the incoming JSON structure
		req := request{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		log.Println("DEBUG: ", req.Email)
		log.Println("DEBUG: ", req.HasVoted)

		// Validate email.
		if req.Email == "" {
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		}

		// Update the user's hasVoted field.
		err := db.UpdateHasVotedByEmail(s.Database.Get(context.Background()), req.Email, req.HasVoted)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return a JSON response of {success: true}
		jsonResponse, err := json.Marshal(response{Success: true})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}
}
