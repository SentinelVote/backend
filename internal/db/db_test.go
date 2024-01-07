package db

import (
	"context"
	"fmt"
	"github.com/alexedwards/argon2id"
	_ "github.com/alexedwards/argon2id"
	"testing"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func TestCreateSchema(t *testing.T) {

	pool, err := sqlitex.Open("file::memory:?mode=memory", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	conn := pool.Get(context.Background())
	defer pool.Put(conn)
	if err := CreateSchema(conn, PRODUCTION, 1); err != nil {
		t.Fatal(err)
	}

	// Query the password of the admin user.
	var passwords []string
	err = sqlitex.Execute(conn, "SELECT password FROM users WHERE email = 'admin@sentinelvote.tech'",
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				passwords = append(passwords, stmt.ColumnText(0))
				return nil
			}})
	if err != nil {
		t.Fatal(err)
	}
	var defaultPasswordHashedQueried = passwords[0]
	t.Log("Query Password : " + defaultPasswordHashedQueried)

	// Verify the password using the Argon2id library.
	match, err := argon2id.ComparePasswordAndHash("password", defaultPasswordHashedQueried)
	if err != nil {
		fmt.Println("Error comparing password and hash : " + err.Error())
	}
	t.Logf("Match: %v", match)

}
