package cmd

import (
	"flag"
	"regexp"
)

type Flags struct {
	URI        string
	Schema     string
	TotalUsers int
}

func ParseCLI() Flags {

	// Define and parse flags.
	uri := flag.String(
		"uri",
		"sqlite3.db",
		"Database URI. Use 'memory', or an alphanumeric filename without extension.",
	)

	schema := flag.String(
		"schema",
		"production",
		"Database schema. Use 'production' or 'simulation' or 'simulation-full'.",
	)

	totalUsers := flag.Int(
		"users",
		3,
		"Number of users to create, a value between 3 and 1000000",
	)

	flag.Parse() // -h and --help is implicitly defined.

	// Validate the database URI.
	if *uri == "memory" || *uri == ":memory:" || *uri == "file::memory:?mode=memory" {
		*uri = "file::memory:?mode=memory"
	} else if *uri != "sqlite3.db" {
		// Check if the string is a valid filename (alphanumeric, underscore, hyphen).
		pattern := `^[a-zA-Z0-9_\-]+$`
		if !regexp.MustCompile(pattern).MatchString(*uri) {
			*uri = "sqlite3.db"
		} else {
			// Append the extension.
			*uri = *uri + ".db"
		}
	}

	// Validate the database schema.
	if *schema != "production" && *schema != "simulation" && *schema != "simulation-full" {
		*schema = "production"
	}

	// Validate the number of users.
	if *totalUsers < 3 || *totalUsers > 1_000_000 {
		*totalUsers = 3
	}

	return Flags{
		URI:        *uri,
		Schema:     *schema,
		TotalUsers: *totalUsers,
	}
}
