package cmd

// Standard library on top, application and third-party packages below.
import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"net/url"

	"backend/internal/db"
	"backend/internal/fabric"
	"backend/internal/lrs"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/zbohm/lirisi/client"
	"github.com/zbohm/lirisi/ring"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// This file is large, thank you for your patience.
// It is organized into sections, each of which is separated by a bordered comment.
// Use your editor's search function to jump to a section:
//
// 1. Models - Structs used in the application
// 1. Authentication Handlers
// 1. General Contract API Functions
// 1. Linkable Ring Signature E-Voting Functions
// 1. Authentication Handlers

// +----------------------------------------------------------------------------------------------+
// |                                            Models                                            |
// +----------------------------------------------------------------------------------------------+

type User struct {
	Email        string `json:"email"`
	FirstName    string `json:"firstName,omitempty"`
	LastName     string `json:"lastName,omitempty"`
	Constituency string `json:"constituency,omitempty"`
	PublicKey    string `json:"publicKey"`
	PrivateKey   string `json:"privateKey,omitempty"`
	HasVoted     bool   `json:"hasVoted,omitempty"`
}

// +----------------------------------------------------------------------------------------------+
// |                                            Helpers                                           |
// +----------------------------------------------------------------------------------------------+

// isHeaderJSON checks if the request header contains the correct content type.
// It writes the error to the response if the header is incorrect.
func isHeaderJSON(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Please send a Content-Type of 'application/json'", http.StatusBadRequest)
		return false
	}
	return true
}

// respondJSON sets the response header to JSON and writes the JSON response.
func respondJSON(w *http.ResponseWriter, response string) {
	(*w).Header().Set("Content-Type", "application/json")
	_, err := (*w).Write([]byte(response))
	if err != nil {
		return
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
		Success             bool   `json:"success"`
		Email               string `json:"email"`
		Constituency        string `json:"constituency"`
		IsCentralAuthority  bool   `json:"isCentralAuthority"`
		HasPublicKey        bool   `json:"hasPublicKey"`
		HasFoldedPublicKeys bool   `json:"hasFoldedPublicKeys"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Check content type and defer resource cleanup.
		if !isHeaderJSON(w, r) {
			return
		}
		defer r.Body.Close()
		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)

		// Match the incoming JSON structure.
		req := request{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate email and password.
		emptyEmail := req.Email == ""
		emptyPassword := req.Password == ""
		if emptyEmail {
			if emptyPassword {
				http.Error(w, "Email and password are required", http.StatusBadRequest)
				return
			}
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		} else if emptyPassword {
			http.Error(w, "Password is required", http.StatusBadRequest)
			return
		}

		// Validate non-empty email string.
		// This mitigates SQL injection attacks.
		if _, err := mail.ParseAddress(req.Email); err != nil {
			http.Error(w, "Invalid email", http.StatusBadRequest)
			return
		}

		// Check if user exists
		if exists, err := db.ExistsUserByEmail(conn, req.Email); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		} else if !exists {
			http.Error(w, "User does not exist", http.StatusUnauthorized)
			return
		}

		var hash string
		var constituency string
		var isCentralAuthority bool
		var publicKey string

		query := `SELECT password, constituency, is_central_authority, public_key FROM users WHERE email = ?;`
		err := sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
			Args: []any{req.Email},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				hash = stmt.ColumnText(0)
				constituency = stmt.ColumnText(1)
				isCentralAuthority = stmt.ColumnBool(2)
				publicKey = stmt.ColumnText(3)
				return nil
			},
		})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Verify password.
		if match, err := argon2id.ComparePasswordAndHash(req.Password, hash); err != nil {
			fmt.Println("Error comparing password and hash : " + err.Error())
		} else if !match {
			http.Error(w, "Invalid password", http.StatusUnauthorized)
			return
		}

		// Check for the existence of the folded public keys, indicating the election has started.
		// This information is used in the frontend to reroute the user to the correct page.
		hasFoldedPublicKeys, err := db.ExistsFoldedPublicKeys(conn)

		// TODO: Optionally set JWT or Paseto authentication token here.

		jsonResponse, err := json.Marshal(response{
			Success:             true,
			Email:               req.Email,
			Constituency:        constituency,
			IsCentralAuthority:  isCentralAuthority,
			HasPublicKey:        publicKey != "",
			HasFoldedPublicKeys: hasFoldedPublicKeys,
		})
		jsonResponseString := string(jsonResponse)
		log.Println("LoginHandler[jsonResponse]: " + jsonResponseString)

		respondJSON(&w, jsonResponseString)
	}
}

// TODO this function is not done or tested.
func (s *Server) handleAuthResetPassword() http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		Response bool `json:"response"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TODO"))
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                     User Table Handlers                                      |
// +----------------------------------------------------------------------------------------------+

func (s *Server) handleGetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)

		var users []User
		err := sqlitex.Execute(conn, "SELECT email, first_name, last_name, constituency, public_key, private_key, has_voted FROM users ORDER BY is_central_authority DESC, (public_key != '') DESC, rowid",
			&sqlitex.ExecOptions{
				ResultFunc: func(stmt *sqlite.Stmt) error {
					user := User{
						Email:        stmt.ColumnText(0),
						FirstName:    stmt.ColumnText(1),
						LastName:     stmt.ColumnText(2),
						Constituency: stmt.ColumnText(3),
						PublicKey:    stmt.ColumnText(4),
						PrivateKey:   stmt.ColumnText(5),
						HasVoted:     stmt.ColumnBool(6),
					}
					users = append(users, user)
					return nil
				},
			})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(users)
		if err != nil {
			http.Error(w, "Error converting users to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}
}

func (s *Server) handleGetUserByEmail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		paramEmail := chi.URLParam(r, "email")
		if paramEmail == "" {
			http.Error(w, "Missing email parameter", http.StatusBadRequest)
			return
		}
		paramEmail, err := url.QueryUnescape(paramEmail)
		if err != nil {
			http.Error(w, "Invalid email parameter", http.StatusBadRequest)
			return
		}
		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)
		log.Println("DEBUG: " + paramEmail)

		var user User
		err = sqlitex.Execute(conn, "SELECT first_name, last_name, constituency, public_key, private_key, has_voted FROM users WHERE email = ? LIMIT 1",
			&sqlitex.ExecOptions{
				Args: []any{paramEmail},
				ResultFunc: func(stmt *sqlite.Stmt) error {
					user = User{
						FirstName:    stmt.ColumnText(0),
						LastName:     stmt.ColumnText(1),
						Constituency: stmt.ColumnText(2),
						PublicKey:    stmt.ColumnText(3),
						PrivateKey:   stmt.ColumnText(4),
						HasVoted:     stmt.ColumnBool(5),
					}
					return nil
				},
			})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse, err := json.Marshal(user)
		if err != nil {
			http.Error(w, "Error converting users to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}
}

func (s *Server) StoreUsersPublicKeysHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check the content type first
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "Please send a 'Content-Type' of 'application/json'", http.StatusBadRequest)
			return
		}

		defer r.Body.Close()

		// Define a request structure to match the incoming JSON structure
		var request struct {
			Email      string `json:"email"`
			PublicKey  string `json:"publicKey"`
			PrivateKey string `json:"privateKey"`
		}

		// Use json.NewDecoder to decode the request body
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}

		// Check for the presence of the email, publicKey, and privateKey parameters
		if request.Email == "" || request.PublicKey == "" || request.PrivateKey == "" {
			http.Error(w, "Missing email, publicKey, or privateKey parameter", http.StatusBadRequest)
			return
		}

		log.Println("STORE_DEBUG_EMAIL:\n" + request.Email)
		log.Println("STORE_DEBUG_PK:\n" + request.PublicKey)
		log.Println("STORE_DEBUG_SK:\n" + request.PrivateKey)

		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)

		// Storing of private keys is only for simulation purposes.
		if err := sqlitex.Execute(conn, "UPDATE users SET public_key = ?, private_key = ? WHERE email = ?",
			&sqlitex.ExecOptions{
				Args: []any{request.PublicKey, request.PrivateKey, request.Email},
			},
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return a JSON response of {success: true}
		response := map[string]string{"message": "Storing of public keys is successful."}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                 Folded Public Keys Handlers                                  |
// +----------------------------------------------------------------------------------------------+

func (s *Server) GetFoldedPublicKeysHandler() http.HandlerFunc {
	type response struct {
		FoldedPublicKeys string `json:"foldedPublicKeys"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)

		foldedPublicKeys, err := db.SelectFoldedPublicKeys(conn)

		jsonResponse, err := json.Marshal(response{
			FoldedPublicKeys: foldedPublicKeys,
		})
		if err != nil {
			http.Error(w, "Error TODO", http.StatusInternalServerError)
			return
		}

		respondJSON(&w, string(jsonResponse))
	}
}

func (s *Server) PutFoldedPublicKeysHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)

		// Fetch every user's public key.
		publicKeys, err := db.GetPublicKeys(conn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Fold public keys.
		foldedPublicKeys, err := lrs.FoldPublicKeys(publicKeys)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Store the folded public keys in the blockchain.
		_, _ = fabric.FabricPutFoldedPublicKeys(foldedPublicKeys)

		// Store the folded public keys in the database.
		err = db.InsertFoldedPublicKeys(conn, foldedPublicKeys)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return a success response.
		other := map[string]bool{
			"sqliteStoreSuccess": true,
			"fabricStoreSuccess": true,
		}
		jsonResponse, err := json.Marshal(other)
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}
}

func (s *Server) ExistsFoldedPublicKeysHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)
		exists, err := sqlitex.ResultBool(conn.Prep("SELECT EXISTS(SELECT folded_public_keys FROM folded_public_keys)"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jsonResponse := fmt.Sprintf(`{"exists":%t}`, exists)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jsonResponse))
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                     Blockchain Handlers                                      |
// +----------------------------------------------------------------------------------------------+

func (s *Server) PutVoteHandler() http.HandlerFunc {
	type request struct {
		Vote         string      `json:"vote"`
		Signature    string      `json:"voteSignature"`
		Constituency string      `json:"constituency"`
		Hour         json.Number `json:"hour"`
	}
	type response struct {
		Success bool `json:"success"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !isHeaderJSON(w, r) {
			return
		}
		defer r.Body.Close()

		// Read the body into a string
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("DEBUG: " + err.Error())
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		bodyString := string(bodyBytes)

		// Define a request structure to match the incoming JSON structure
		req := request{}
		if err = json.Unmarshal(bodyBytes, &req); err != nil {
			log.Println("DEBUG: " + err.Error())
			http.Error(w, "Error parsing JSON request body", http.StatusBadRequest)
			return
		}

		// TODO:
		//  Optionally validate the vote and signature.
		//  Validation has already been implemented in Fabric.

		// Store the vote in the blockchain using the original JSON string
		uuidv7, _ := uuid.NewV7()
		_, err = fabric.FabricPutVote(uuidv7.String(), bodyString)

		if err != nil {
			http.Error(w, "Error storing vote", http.StatusInternalServerError)
			return
		}

		// Send a successful response
		jsonResponse, err := json.Marshal(response{Success: true})
		if err != nil {
			http.Error(w, "Error converting response to JSON", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jsonResponse)
	}
}

// +----------------------------------------------------------------------------------------------+
// |                               Linkable Ring Signature Handlers                               |
// +----------------------------------------------------------------------------------------------+

// handleGenerateKeys generates a private key and a public key.
func (s *Server) handleGenerateKeys() http.HandlerFunc {
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

		// Write the JSON to the response
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	}
}

// handleSign creates a linkable ring signature.
func (s *Server) handleSign() http.HandlerFunc {
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
		defer r.Body.Close()

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

		w.Write(jsonResponse)
	}
}
