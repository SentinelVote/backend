package db

import (
	"backend/internal/foldpub"
	"log"
	"os"
	"strconv"
	"strings"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

//goland:noinspection GoSnakeCaseUsage,GoCommentStart
const (
	// Public keys and private keys are not initialized in the database.
	PRODUCTION = iota

	// Public keys and private keys are initialized in the database.
	SIMULATION

	// Public keys and private keys are initialized in the database,
	// and folded public keys are inserted into the blockchain.
	SIMULATION_FULL
)

func CreateSchema(conn *sqlite.Conn, purpose int, totalUsers int) error {

	log.Println("Creating schema...")

	// Perform a string replacement to insert our chosen number of users.
	// We also minus two, because we inserted 2 users from InsertDefault.
	var insertMany string
	if purpose == PRODUCTION {
		insertMany = strings.ReplaceAll(InsertProduction, "?1", strconv.Itoa(totalUsers-2))
	} else {
		insertMany = strings.ReplaceAll(InsertSimulation, "?1", strconv.Itoa(totalUsers-2))
	}

	// The SQL transaction string to be executed.
	// BEGIN TRANSACTION and COMMIT is implicitly done by sqlitex.ExecScript.
	sep := "\n"
	var transaction = strings.Join([]string{
		Schema,
		Constituencies,
		FirstNames,
		LastNames,
		InsertDefault,
		insertMany,
	}, sep)

	// Write the query string to disk (for debugging purposes).
	if err := os.WriteFile("public/query.sql", []byte(transaction), os.FileMode(0644)); err != nil {
		return err
	}

	log.Println("Executing SQL transaction...")
	if err := sqlitex.ExecScript(conn, transaction); err != nil {
		return err
	}
	log.Println("Successfully executed SQL transaction.")

	// Write the PEM files to disk (for debugging purposes).
	if err := writeKeys(conn); err != nil {
		return err
	}

	// A full simulation will also store the folded public keys in the blockchain.
	if purpose == SIMULATION_FULL {
		if response, err := foldpub.PutFoldedPublicKeys(conn); err != nil {
			log.Println("Unable to insert folded public keys into the blockchain.")
			log.Println("Error message: " + err.Error())
		} else if response != "OK" {
			log.Println("Unable to insert folded public keys into the blockchain.")
			log.Println("Error message: " + response)
		} else {
			log.Println("Successfully inserted folded public keys into the blockchain.")
		}
	}

	return nil
}

func writeKeys(conn *sqlite.Conn) error {

	var query string
	perm := os.FileMode(0644)

	query = "SELECT public_key FROM users WHERE ROWID = 2;"
	publicKeyUser1, err := sqlitex.ResultText(conn.Prep(query))
	if err != nil {
		return err
	}
	err = os.WriteFile("public/publicKeyUser1.pem", []byte(publicKeyUser1), perm)
	if err != nil {
		return err
	}

	query = "SELECT public_key FROM users WHERE ROWID = 3;"
	publicKeyUser2, err := sqlitex.ResultText(conn.Prep(query))
	if err != nil {
		return err
	}
	err = os.WriteFile("public/publicKeyUser2.pem", []byte(publicKeyUser2), perm)
	if err != nil {
		return err
	}

	query = "SELECT private_key FROM users WHERE ROWID = 2;"
	privateKeyUser1, err := sqlitex.ResultText(conn.Prep(query))
	if err != nil {
		return err
	}
	err = os.WriteFile("public/privateKeyUser1.pem", []byte(privateKeyUser1), perm)
	if err != nil {
		return err
	}

	query = "SELECT private_key FROM users WHERE ROWID = 3;"
	privateKeyUser2, err := sqlitex.ResultText(conn.Prep(query))
	if err != nil {
		return err
	}
	err = os.WriteFile("public/privateKeyUser2.pem", []byte(privateKeyUser2), perm)
	if err != nil {
		return err
	}

	return nil
}
