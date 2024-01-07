package db

import (
	"log"
	"net/mail"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// This file stores queries and their respective wrapper functions.

// +--------------------------------------------------------------------------+
// |                         TABLE is_end_of_election                         |
// +--------------------------------------------------------------------------+

// ExistsIsEndOfElection returns true if there exists a row in the is end of election table.
func ExistsIsEndOfElection(conn *sqlite.Conn) (bool, error) {
	log.Println("Called: app/db/queries.go: ExistsIsEndOfElection()")

	exists, err := sqlitex.ResultBool(conn.Prep(`SELECT EXISTS(SELECT is_end_of_election FROM is_end_of_election);`))
	if err != nil {
		return false, err
	}

	return exists, nil
}

// InsertIsEndOfElection inserts a row into the table, indicating that the election has ended.
func InsertIsEndOfElection(conn *sqlite.Conn, isEndOfElection bool) error {
	log.Println("Called: app/db/queries.go: InsertIsEndOfElection()")
	return sqlitex.Execute(conn, `INSERT INTO is_end_of_election (singleton, is_end_of_election) VALUES (1, true);`, nil)
}

// +--------------------------------------------------------------------------+
// |                         TABLE folded_public_keys                         |
// +--------------------------------------------------------------------------+

// ExistsFoldedPublicKeys returns true if there exists a row in the folded public keys table.
func ExistsFoldedPublicKeys(conn *sqlite.Conn) (bool, error) {
	log.Println("Called: app/db/queries.go: ExistsFoldedPublicKeys()")

	exists, err := sqlitex.ResultBool(conn.Prep(`SELECT EXISTS(SELECT folded_public_keys FROM folded_public_keys);`))
	if err != nil {
		return false, err
	}

	return exists, nil
}

// SelectFoldedPublicKeys returns the folded public keys (linkable ring signature group).
func SelectFoldedPublicKeys(conn *sqlite.Conn) (string, error) {
	log.Println("Called: app/db/queries.go: SelectFoldedPublicKeys()")

	var foldedPublicKeys string
	err := sqlitex.Execute(conn, `SELECT folded_public_keys FROM folded_public_keys LIMIT 1;`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			foldedPublicKeys = stmt.ColumnText(0)
			return nil
		},
	})
	if err != nil {
		return "", err
	}

	return foldedPublicKeys, nil
}

// InsertFoldedPublicKeys inserts the folded public keys (linkable ring signature group).
func InsertFoldedPublicKeys(conn *sqlite.Conn, foldedPublicKeys string) error {
	log.Println("Called: app/db/queries.go: InsertFoldedPublicKeys()")

	return sqlitex.Execute(conn, `INSERT INTO folded_public_keys (singleton, folded_public_keys) VALUES (1, ?);`, &sqlitex.ExecOptions{
		Args: []any{foldedPublicKeys},
	})
}

// +--------------------------------------------------------------------------+
// |                               TABLE users                                |
// +--------------------------------------------------------------------------+

// ExistsUserByEmail returns true if the user exists.
func ExistsUserByEmail(conn *sqlite.Conn, email string) (bool, error) {
	log.Println("Called: app/db/queries.go: ExistsUserByEmail()")

	if _, err := mail.ParseAddress(email); err != nil {
		// Email string is not a valid email address.
		return false, err
	}
	var exists bool
	err := sqlitex.Execute(conn, `SELECT EXISTS(SELECT 1 FROM users WHERE email = ?);`, &sqlitex.ExecOptions{
		Args: []any{email},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			exists = stmt.ColumnBool(0)
			return nil
		},
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}

// GetPublicKeys returns the public keys of all users.
func GetPublicKeys(conn *sqlite.Conn) ([]string, error) {
	log.Println("Called: app/db/queries.go: GetPublicKeys()")

	var publicKeys []string
	err := sqlitex.Execute(conn, `SELECT public_key FROM users WHERE is_central_authority = FALSE AND public_key != '';`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			publicKeys = append(publicKeys, stmt.ColumnText(0))
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return publicKeys, nil
}

// UpdateHasVotedByEmail returns the public keys of all users.
func UpdateHasVotedByEmail(conn *sqlite.Conn, email string, hasVoted bool) error {
	log.Println("Called: app/db/queries.go: UpdateHasVotedByEmail()")
	if _, err := mail.ParseAddress(email); err != nil {
		// Email string is not a valid email address.
		return err
	}
	return sqlitex.Execute(conn, `UPDATE users SET has_voted = ? WHERE email = ?;`, &sqlitex.ExecOptions{
		Args: []any{hasVoted, email},
	})
}
