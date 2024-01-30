package db

import (
	"log"
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
