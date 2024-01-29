package cmd

import (
	"context"
	"github.com/goccy/go-json"
	"net/http"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// +----------------------------------------------------------------------------------------------+
// |                                          Admin Handlers                                      |
// +----------------------------------------------------------------------------------------------+

func (s *Server) handleAdminGetUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(context.Background())
		defer s.Database.Put(conn)

		var users []User
		err := sqlitex.Execute(conn, "SELECT email, first_name, last_name, constituency, public_key, private_key, has_voted FROM users WHERE is_central_authority = FALSE ORDER BY (public_key != '') DESC, rowid",
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
