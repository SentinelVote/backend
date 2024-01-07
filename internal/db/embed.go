package db

import _ "embed"

//go:embed migration/schema.sql
var Schema string

//go:embed data/constituencies.sql
var Constituencies string

//go:embed data/first_names.sql
var FirstNames string

//go:embed data/last_names.sql
var LastNames string

// InsertDefault inserts two users with public and private keys initialized.
// This mitigates the minimum number of public keys required in ring.MakeSignature
//
//go:embed insert_default.sql
var InsertDefault string

//go:embed insert_production.sql
var InsertProduction string

//go:embed insert_simulation.sql
var InsertSimulation string
