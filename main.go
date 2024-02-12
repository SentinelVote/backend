///usr/bin/env go run "$0" "$@" ; exit "$?"
package main

import (
	"github.com/sentinelvote/backend/cmd"
	"log"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running the application: %v", err)
	}
}
