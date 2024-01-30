package cmd

import (
	"github.com/go-chi/chi/v5"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Server struct {
	Router     *chi.Mux
	Database   *sqlitex.Pool
	URI        string // Filepath to the database, or `:memory:`
	TotalUsers int    // Number of users to create
	PoolSize   int    // Number of connections to the database
	Schema     string // `production` or `simulation` or `simulation_full`
}
