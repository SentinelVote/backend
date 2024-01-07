package db

import (
	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/zbohm/lirisi/client"
	"github.com/zbohm/lirisi/ring"
	"zombiezen.com/go/sqlite"
)

// This file registers SQLite functions.
// See https://sqlite.org/appfunc.html for more information.

// SQLiteFunctionPrivateKey registers an SQLite function that generates a private key.
func SQLiteFunctionPrivateKey(conn *sqlite.Conn) error {
	err := conn.CreateFunction("generate_private_key", &sqlite.FunctionImpl{
		NArgs:         0,
		Deterministic: false,
		AllowIndirect: true,
		Scalar: func(ctx sqlite.Context, args []sqlite.Value) (sqlite.Value, error) {
			status, privateKey := client.GeneratePrivateKey("prime256v1", "PEM")
			if status != ring.Success {
				return sqlite.IntegerValue(int64(status)), nil
			}
			return sqlite.TextValue(string(privateKey)), nil
		},
	})
	return err
}

// SQLiteFunctionPublicKey registers an SQLite function that derives a public key from a private key.
func SQLiteFunctionPublicKey(conn *sqlite.Conn) error {
	err := conn.CreateFunction("derive_public_key", &sqlite.FunctionImpl{
		NArgs:         1,
		Deterministic: false,
		AllowIndirect: true,
		Scalar: func(ctx sqlite.Context, args []sqlite.Value) (sqlite.Value, error) {
			status, publicKey := client.DerivePublicKey([]byte(args[0].Text()), "PEM")
			if status != ring.Success {
				return sqlite.IntegerValue(int64(status)), nil
			}
			return sqlite.TextValue(string(publicKey)), nil
		},
	})
	return err
}

// SQLiteFunctionArgon2id registers an SQLite function that hashes a password.
func SQLiteFunctionArgon2id(conn *sqlite.Conn) error {
	err := conn.CreateFunction("argon2id", &sqlite.FunctionImpl{
		NArgs:         1,
		Deterministic: false,
		AllowIndirect: true,
		Scalar: func(ctx sqlite.Context, args []sqlite.Value) (sqlite.Value, error) {
			hash, err := argon2id.CreateHash(args[0].Text(), argon2id.DefaultParams)
			if err != nil {
				return sqlite.TextValue(err.Error()), nil
			}
			return sqlite.TextValue(hash), nil
		},
	})
	return err
}

// SQLiteFunctionUUIDv7 registers an SQLite function that generates a UUIDv7.
func SQLiteFunctionUUIDv7(conn *sqlite.Conn) error {
	err := conn.CreateFunction("uuidv7", &sqlite.FunctionImpl{
		NArgs:         0,
		Deterministic: false,
		AllowIndirect: true,
		Scalar: func(ctx sqlite.Context, args []sqlite.Value) (sqlite.Value, error) {
			uuidv7, err := uuid.NewV7()
			if err != nil {
				return sqlite.TextValue(err.Error()), nil
			}
			return sqlite.TextValue(uuidv7.String()), nil
		},
	})
	return err
}
