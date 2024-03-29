package cmd

// Standard library on top, application and third-party packages below.
import (
	"io"
	"log"
	"net/http"
	"net/mail"

	"github.com/alexedwards/argon2id"
	"github.com/goccy/go-json"
	"github.com/sentinelvote/backend/internal/foldpub"
	"github.com/zbohm/lirisi/client"
	"github.com/zbohm/lirisi/ring"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// This file organized into sections, each of which is separated by a bordered comment.
// Use your editor's search function to jump to a section:
//
// Helpers
// Authentication Handlers
// Admin and Voter Handlers
// Admin Handlers
// Voter Handlers

// +----------------------------------------------------------------------------------------------+
// |                                           Helpers                                            |
// +----------------------------------------------------------------------------------------------+

// isHeaderJSON checks if the request header contains the correct content type.
// It writes the error to the response if the header is incorrect.
func isHeaderJSON(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		log.Println("Error: Please send a 'Content-Type' of 'application/json'")
		http.Error(w, "Please send a Content-Type of 'application/json'", http.StatusBadRequest)
		return false
	}
	return true
}

// respondJSON sets the response header to JSON and writes the JSON response.
// T is a type parameter constrained to types that can be converted to []byte.
func respondJSON[T string | []byte](w *http.ResponseWriter, response T) {
	(*w).Header().Set("Content-Type", "application/json")
	_, err := (*w).Write([]byte(response))
	if err != nil {
		log.Println("Error writing response: " + err.Error())
		return
	}
}

// respondPlainText sets the response header to plain text and writes the plain text response.
// T is a type parameter constrained to types that can be converted to []byte.
func respondPlainText[T string | []byte](w *http.ResponseWriter, response T) {
	(*w).Header().Set("Content-Type", "text/plain")
	_, err := (*w).Write([]byte(response))
	if err != nil {
		log.Println("Error writing response: " + err.Error())
		return
	}
}

// bodyClose closes the request body and logs any errors.
func bodyClose(Body io.ReadCloser) {
	err := Body.Close()
	if err != nil {
		log.Println("Error closing request body: " + err.Error())
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                   Authentication Handlers                                    |
// +----------------------------------------------------------------------------------------------+

// handleAuthLogin handles login requests.
func (s *Server) handleAuthLogin() http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		Success            bool   `json:"success"`
		Email              string `json:"email"`
		Constituency       string `json:"constituency"`
		IsCentralAuthority bool   `json:"isCentralAuthority"`
		HasPublicKey       bool   `json:"hasPublicKey"`
		HasVoted           bool   `json:"hasVoted"`
		HasDefaultPassword bool   `json:"hasDefaultPassword"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !isHeaderJSON(w, r) {
			return
		}
		defer bodyClose(r.Body)
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Match the incoming JSON structure.
		req := request{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate email string (mitigates SQL injection).
		if _, err := mail.ParseAddress(req.Email); err != nil || req.Email == "" {
			http.Error(w, "Invalid email or password", http.StatusBadRequest)
			return
		}

		var email string
		var hash string
		var constituency string
		var isCentralAuthority bool
		var publicKey string
		var hasVoted bool
		var hasDefaultPassword bool

		query := `SELECT password, constituency, is_central_authority, public_key, has_voted, has_default_password, email FROM users WHERE email = ?;`
		err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
			Args: []any{req.Email},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				hash = stmt.ColumnText(0)
				constituency = stmt.ColumnText(1)
				isCentralAuthority = stmt.ColumnBool(2)
				publicKey = stmt.ColumnText(3)
				hasVoted = stmt.ColumnBool(4)
				hasDefaultPassword = stmt.ColumnBool(5)
				email = stmt.ColumnText(6)
				return nil
			},
		})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If the user does not exist, return an error.
		if email == "" || email != req.Email {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Verify password.
		if match, err := argon2id.ComparePasswordAndHash(req.Password, hash); err != nil {
			log.Println("Error comparing password and hash : " + err.Error())
		} else if !match {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		jsonResponse, err := json.Marshal(response{
			Email:              req.Email,
			Constituency:       constituency,
			IsCentralAuthority: isCentralAuthority,
			HasPublicKey:       publicKey != "",
			HasVoted:           hasVoted,
			HasDefaultPassword: hasDefaultPassword,
		})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}

		respondJSON(&w, jsonResponse)
	}
}

func (s *Server) handleAuthResetPassword() http.HandlerFunc {
	type request struct {
		Email string `json:"email"`
	}
	type response struct {
		Response bool `json:"response"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !isHeaderJSON(w, r) {
			return
		}
		defer bodyClose(r.Body)
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Match the incoming JSON structure.
		req := request{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Check if the user exists, and the user is not a central authority.
		// Central authority should not reset their password from the frontend interface.
		var email string
		query := `SELECT email FROM users WHERE email = ? AND is_central_authority = FALSE;`
		err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
			Args: []any{req.Email},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				email = stmt.ColumnText(0)
				return nil
			},
		})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if email == "" || email != req.Email {
			http.Error(w, "Invalid email", http.StatusBadRequest)
			return
		}

		// Use the helper function to update the user's password.
		err = helperUpdatePassword(conn, req.Email, "password")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(response{Response: true})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		respondJSON(&w, jsonResponse)
	}
}

func (s *Server) handleAuthUpdatePassword() http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		Response bool `json:"response"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !isHeaderJSON(w, r) {
			return
		}
		defer bodyClose(r.Body)
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Match the incoming JSON structure.
		req := request{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Use the helper function to update the user's password.
		err := helperUpdatePassword(conn, req.Email, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set the has_default_password field to false.
		err = sqlitex.Execute(conn, `UPDATE users SET has_default_password = FALSE WHERE email = ?;`, &sqlitex.ExecOptions{
			Args: []any{req.Email},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(response{Response: true})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		respondJSON(&w, jsonResponse)
	}
}

func helperUpdatePassword(conn *sqlite.Conn, email string, newPassword string) error {
	// Validate email string (mitigates SQL injection).
	if _, err := mail.ParseAddress(email); err != nil {
		return err
	}

	// Hash the password.
	newHash, err := argon2id.CreateHash(newPassword, argon2id.DefaultParams)
	if err != nil {
		return err
	}

	// Update the user's password.
	query := `UPDATE users SET password = ? WHERE email = ?;`
	return sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []any{newHash, email},
	})
}

// +----------------------------------------------------------------------------------------------+
// |                                   Admin and Voter Handlers                                   |
// +----------------------------------------------------------------------------------------------+

func (s *Server) handleIsEndOfElection() http.HandlerFunc {
	query := `SELECT EXISTS (SELECT 1 FROM is_end_of_election);`
	var response string

	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		var isEndOfElection bool
		err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				isEndOfElection = stmt.ColumnBool(0)
				return nil
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if isEndOfElection {
			response = "true"
		} else {
			response = "false"
		}
		respondPlainText(&w, response)
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                        Admin Handlers                                        |
// +----------------------------------------------------------------------------------------------+

func (s *Server) handleAdminGetUsers() http.HandlerFunc {
	const query = `
		SELECT json_group_array(json_object(
			'email', email,
			'firstName', first_name,
			'lastName', last_name,
			'constituency', constituency,
			'publicKey', public_key,
			'privateKey', private_key,
			'hasVoted', has_voted
		)) as result
		FROM users
		WHERE is_central_authority = FALSE
		ORDER BY (public_key != '') DESC, rowid;`

	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		jsonResponse, err := sqlitex.ResultText(conn.Prep(query))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respondJSON(&w, jsonResponse)
	}
}

func (s *Server) handleAdminPutFoldedPublicKeys() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Store the folded public keys in the blockchain.
		message, err := foldpub.PutFoldedPublicKeys(conn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondPlainText(&w, message)
	}
}

func (s *Server) handleAdminAnnounceResult() http.HandlerFunc {
	query := `INSERT INTO is_end_of_election  VALUES (1);`
	expected := "sqlite: step: constraint failed: UNIQUE constraint failed: is_end_of_election.is_end_of_election"

	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		err := sqlitex.Execute(conn, query, nil)
		if err != nil {
			if err.Error() == expected {
				respondPlainText(&w, "Already inserted into is_end_of_election, not inserting again")
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		respondPlainText(&w, "Successfully inserted into is_end_of_election")
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                        Voter Handlers                                        |
// +----------------------------------------------------------------------------------------------+

// handleVoterGenerateKeys generates a private key and a public key.
func (s *Server) handleVoterGenerateKeys() http.HandlerFunc {
	type response struct {
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		status, privateKey := client.GeneratePrivateKey("prime256v1", "PEM")
		if status != ring.Success {
			http.Error(w, ring.ErrorMessages[status], http.StatusInternalServerError)
			return
		}
		status, publicKey := client.DerivePublicKey(privateKey, "PEM")
		if status != ring.Success {
			http.Error(w, ring.ErrorMessages[status], http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(response{
			PublicKey:  string(publicKey),
			PrivateKey: string(privateKey),
		})
		if err != nil {
			http.Error(w, "Error converting keys to JSON", http.StatusInternalServerError)
			return
		}

		respondJSON(&w, jsonResponse)
	}
}

// handleVoterSign creates a linkable ring signature.
func (s *Server) handleVoterSign() http.HandlerFunc {
	type request struct {
		FoldedPublicKeys  string `json:"foldedPublicKeys"`
		PrivateKeyContent string `json:"privateKeyContent"`
		Message           string `json:"message"`
	}
	type response struct {
		Signature string `json:"signature"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Please send a 'Content-Type' of 'application/json'", http.StatusBadRequest)
			return
		}
		defer bodyClose(r.Body)

		// Define a request structure to match the incoming JSON structure
		req := request{}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Println("DEBUG: " + err.Error())
			http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}

		// Validate required parameters.
		if req.FoldedPublicKeys == "" {
			http.Error(w, "Missing foldedPublicKeys parameter", http.StatusBadRequest)
			return
		}
		if req.PrivateKeyContent == "" {
			http.Error(w, "Missing privateKeyContent parameter", http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, "Missing message parameter", http.StatusBadRequest)
			return
		}

		// Convert JSON fields to byte arrays.
		foldedPublicKeys := []byte(req.FoldedPublicKeys)
		privateKeyContent := []byte(req.PrivateKeyContent)
		message := []byte(req.Message)

		// Sign message. caseIdentifier is empty because there's no multi-round voting.
		status, signature := client.CreateSignature(foldedPublicKeys, privateKeyContent, message, []byte(""), "PEM")
		if status != ring.Success {
			http.Error(w, ring.ErrorMessages[status], http.StatusInternalServerError)
			return
		}

		// Convert the response format to JSON, consisting of "signature" field.
		jsonResponse, err := json.Marshal(response{
			Signature: string(signature),
		})
		if err != nil {
			http.Error(w, "Error converting signature to JSON", http.StatusInternalServerError)
			return
		}

		respondJSON(&w, jsonResponse)
	}
}

// handleVoterUpdateHasVotedByEmail updates the has_voted database field of a user.
func (s *Server) handleVoterUpdateHasVotedByEmail() http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		HasVoted bool   `json:"hasVoted"`
	}
	type response struct {
		Success bool `json:"success"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Called: handleVoterUpdateHasVotedByEmail()")
		if !isHeaderJSON(w, r) {
			return
		}
		defer bodyClose(r.Body)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Validate email.
		if _, err := mail.ParseAddress(req.Email); err != nil {
			http.Error(w, "Invalid email", http.StatusBadRequest)
			return
		}

		// Update the user's hasVoted field.
		err := sqlitex.Execute(conn, `UPDATE users SET has_voted = ? WHERE email = ?;`, &sqlitex.ExecOptions{
			Args: []any{req.HasVoted, req.Email},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(response{Success: true})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		respondJSON(&w, jsonResponse)
	}
}

// handleVoterUpdateKeysByEmail updates the public key and private key of a user.
func (s *Server) handleVoterUpdateKeysByEmail() http.HandlerFunc {
	type request struct {
		Email      string `json:"email"`
		PublicKey  string `json:"publicKey"`
		PrivateKey string `json:"privateKey"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Called: handleVoterUpdateKeysByEmail()")
		if !isHeaderJSON(w, r) {
			return
		}
		defer bodyClose(r.Body)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Validate email.
		if _, err := mail.ParseAddress(req.Email); err != nil {
			http.Error(w, "Invalid email", http.StatusBadRequest)
			return
		}

		// Store the public key.
		if req.PublicKey == "" {
			http.Error(w, "Missing publicKey parameter", http.StatusBadRequest)
			return
		}
		if err := sqlitex.Execute(conn, "UPDATE users SET public_key = ? WHERE email = ?",
			&sqlitex.ExecOptions{
				Args: []any{req.PublicKey, req.Email},
			},
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Store the private key (for simulation purposes).
		if req.PrivateKey != "" {
			if err := sqlitex.Execute(conn, "UPDATE users SET private_key = ? WHERE email = ?",
				&sqlitex.ExecOptions{
					Args: []any{req.PrivateKey, req.Email},
				},
			); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Return a JSON response of {success: true}
		response := map[string]string{"message": "Storing of public keys is successful."}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		respondJSON(&w, jsonResponse)
	}
}

// handleVoterGetPrivateKeyByEmail returns the private key of a user (for simulation purposes).
func (s *Server) handleVoterGetPrivateKeyByEmail() http.HandlerFunc {
	type request struct {
		Email string `json:"email"`
	}
	type response struct {
		PrivateKey string `json:"privateKey"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Called: handleVoterGetPrivateKeyByEmail()")
		if !isHeaderJSON(w, r) {
			return
		}
		defer bodyClose(r.Body)
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		// Validate email.
		if _, err := mail.ParseAddress(req.Email); err != nil {
			http.Error(w, "Invalid email", http.StatusBadRequest)
			return
		}

		// Get the private key of the user.
		var privateKey string
		if err := sqlitex.Execute(conn,
			"SELECT private_key FROM users WHERE email = ?",
			&sqlitex.ExecOptions{
				Args: []any{req.Email},
				ResultFunc: func(stmt *sqlite.Stmt) error {
					privateKey = stmt.ColumnText(0)
					return nil
				},
			},
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(response{PrivateKey: privateKey})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		respondJSON(&w, jsonResponse)
	}
}
