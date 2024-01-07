package db

import (
	"backend/internal/fabric"
	"backend/internal/lrs"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/alexedwards/argon2id"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const (
	PRODUCTION      = iota // Public keys, private keys, and folded public keys are not initialized in the database.
	SIMULATION             // Public keys, private keys, and folded public keys are initialized in the database.
	SIMULATION_FULL        // Public keys, private keys, and folded public keys are initialized in the database, and folded public keys are inserted into the blockchain.
)

func CreateSchema(conn *sqlite.Conn, purpose int, initialUserCount int) error {
	log.Println("Creating schema...")
	var err error = nil

	// Define a non-default password for the admin, user1, and user2.
	// Hashing is done via argon2id() in insert_default.sql.
	nonDefaultPassword := "Password1!"
	log.Println("Non-default password (actual): " + nonDefaultPassword)

	// Define a default password, for the remaining users.
	defaultPassword := "password"
	// Hash the password now; the query string will be replaced later.
	// Everyone uses the same hash,
	// because doing so for a large number of users is computationally expensive.
	defaultPasswordHashed, err := argon2id.CreateHash(defaultPassword, argon2id.DefaultParams)
	if err != nil {
		return err
	}
	log.Println("Default password (actual): " + defaultPassword)
	log.Println("Default password (hashed): " + defaultPasswordHashed)

	var dbInsertMany string
	if purpose == PRODUCTION {
		dbInsertMany = InsertProduction
	} else {
		dbInsertMany = InsertSimulation
	}

	// Replace the query string's password with the hashed password.
	dbInsertDefault := strings.ReplaceAll(InsertDefault, "'password'", "'"+nonDefaultPassword+"'")
	dbInsertMany = strings.ReplaceAll(dbInsertMany, "'password'", "'"+defaultPasswordHashed+"'")

	// We inserted 2 users in dbInsertDefault.
	initialUserCount -= 2

	// Replace the query string's initial number of users with the chosen number of users.
	replacedValue := "LIMIT " + strconv.Itoa(initialUserCount)
	dbInsertMany = strings.ReplaceAll(dbInsertMany, "LIMIT 8", replacedValue)

	// The SQL transaction string to be executed.
	// BEGIN TRANSACTION and COMMIT is done by sqlitex.ExecScript.
	sep := "\n"
	var transaction = strings.Join([]string{
		Schema,
		Constituencies,
		FirstNames,
		LastNames,
		dbInsertDefault,
		dbInsertMany,
	}, sep)
	// log.Println("SQL transaction string after replacements:", transaction)

	// Execute the SQL transaction.
	log.Println("Executing SQL transaction...")
	if err := sqlitex.ExecScript(conn, transaction); err != nil {
		return err
	}
	log.Println("Successfully executed SQL transaction.")

	// Write the PEM files to disk.
	if err := writeKeys(conn); err != nil {
		return err
	}

	// Generate key pairs and the linkable ring signature group.
	if purpose == SIMULATION || purpose == SIMULATION_FULL {
		if publicKeys, err := GetPublicKeys(conn); err != nil {
			return err
		} else if foldedPublicKeys, err := lrs.FoldPublicKeys(publicKeys); err != nil {
			return err
		} else if err = InsertFoldedPublicKeys(conn, foldedPublicKeys); err != nil {
			return err
		} else if purpose == SIMULATION_FULL {
			if res, err := fabric.FabricPutFoldedPublicKeys(foldedPublicKeys); err != nil {
				log.Println("Unable to insert folded public keys into the blockchain.")
				log.Println("Error message: " + err.Error())
			} else if res == "OK" {
				log.Println("Successfully inserted folded public keys into the blockchain.")
			} else {
				log.Println("Unable to insert folded public keys into the blockchain.")
				log.Println("Error message: " + res)
			}
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
