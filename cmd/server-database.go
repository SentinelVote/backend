package cmd

// Standard library on top, third-party packages below.
import (
	"backend/internal/db"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func (s *Server) database(uri string) error {
	// Handle existing database files.
	if uri != "file::memory:?mode=memory" {
		uri = filepath.Join("public", uri)
		if err := removeDatabaseFileIfExists(uri); err != nil {
			return err
		}
		if err := removeDatabaseFileIfExists(uri + "-shm"); err != nil {
			return err
		}
		if err := removeDatabaseFileIfExists(uri + "-wal"); err != nil {
			return err
		}
	}

	// Create a new database.
	if pool, err := sqlitex.NewPool(uri, sqlitex.PoolOptions{
		Flags:    0,
		PoolSize: s.PoolSize,
		PrepareConn: func(conn *sqlite.Conn) error {
			if err := db.SQLiteFunctionArgon2id(conn); err != nil {
				return err
			}
			if err := db.SQLiteFunctionPrivateKey(conn); err != nil {
				return err
			}
			if err := db.SQLiteFunctionPublicKey(conn); err != nil {
				return err
			}
			if err := db.SQLiteFunctionUUIDv7(conn); err != nil {
				return err
			}
			return nil
		},
	}); err != nil {
		return err
	} else {
		log.Println("Created new database at " + uri)
		s.Database = pool
	}
	conn := s.Database.Get(context.Background())
	defer s.Database.Put(conn)

	// Set up schema parameters.
	if s.Schema == "production" {
		return db.CreateSchema(conn, db.PRODUCTION, s.TotalUsers)
	} else if s.Schema == "simulation" {
		return db.CreateSchema(conn, db.SIMULATION, s.TotalUsers)
	} else if s.Schema == "simulation-full" {
		return db.CreateSchema(conn, db.SIMULATION_FULL, s.TotalUsers)
	} else {
		return fmt.Errorf("👻") // This should never happen.
	}
}

func removeDatabaseFileIfExists(filename string) error {
	if _, err := os.Stat(filename); err == nil {
		log.Printf("Found existing database file at `%s`, removing...\n", filename)
		if err := os.Remove(filename); err != nil {
			log.Printf("Failed to remove existing database file at `%s`.\n", filename)
			return err
		}
	}
	return nil
}
