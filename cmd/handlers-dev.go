package cmd

// Standard library on top, third-party packages below.
import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"backend/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"zombiezen.com/go/sqlite/sqlitex"
)

// THE HANDLERS USED IN THIS FILE ARE ONLY INTENDED FOR DEVELOPMENT PURPOSES.

// +----------------------------------------------------------------------------------------------+
// |                                            System                                            |
// +----------------------------------------------------------------------------------------------+

// handleDevPanic helps in testing middleware.Recoverer error recovery and logging mechanisms.
func (s *Server) handleDevPanic() http.HandlerFunc {
	return func(_ http.ResponseWriter, _ *http.Request) {
		panic("Test panic called by handleDevPanic()")
	}
}

// handleDevMemApp reports the application's memory usage in MB
func (s *Server) handleDevMemApp() http.HandlerFunc {
	type response struct {
		Unit       string      `json:"unit"`
		Alloc      json.Number `json:"alloc"`       // MB allocated and still in use.
		TotalAlloc json.Number `json:"total_alloc"` // Total MB allocated (even if freed).
		Sys        json.Number `json:"sys"`         // MB obtained from the system.
		NumGC      json.Number `json:"num_gc"`      // Number of completed GC cycles.
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Convert bytes to MB
		allocMB := float64(m.Alloc) / 1024 / 1024
		totalAllocMB := float64(m.TotalAlloc) / 1024 / 1024
		sysMB := float64(m.Sys) / 1024 / 1024

		jsonResponse, err := json.Marshal(response{
			Unit:       "MB",
			Alloc:      json.Number(strconv.FormatFloat(allocMB, 'f', 2, 64)),
			TotalAlloc: json.Number(strconv.FormatFloat(totalAllocMB, 'f', 2, 64)),
			Sys:        json.Number(strconv.FormatFloat(sysMB, 'f', 2, 64)),
			NumGC:      json.Number(strconv.Itoa(int(m.NumGC))),
		})
		if err != nil {
			http.Error(w, "Error converting app mem info to JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonResponse)
		if err != nil {
			log.Println(err)
			http.Error(w, "Error writing response", http.StatusInternalServerError)
		}
	}
}

// handleDevMemSystem reports the system's memory usage in MB.
func (s *Server) handleDevMemSystem() http.HandlerFunc {
	type response struct {
		Unit  string      `json:"unit"`
		Total json.Number `json:"total"`
		Used  json.Number `json:"used"`
		Free  json.Number `json:"free"`
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		memInfo, err := meminfoParse()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading memory info: %v", err), http.StatusInternalServerError)
			return
		}

		total, totalOk := memInfo["MemTotal"]
		if !totalOk {
			http.Error(w, "Failed to parse total memory info", http.StatusInternalServerError)
			return
		}

		free, freeOk := memInfo["MemFree"]
		if !freeOk {
			http.Error(w, "Failed to parse free memory info", http.StatusInternalServerError)
			return
		}
		used := total - free

		jsonResponse, err := json.Marshal(response{
			Unit:  "MB",
			Total: json.Number(strconv.Itoa(total)),
			Used:  json.Number(strconv.Itoa(used)),
			Free:  json.Number(strconv.Itoa(free)),
		})
		if err != nil {
			http.Error(w, "Error converting sys mem info to JSON", http.StatusInternalServerError)
			return
		}

		// Write the JSON to the response
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonResponse)
		if err != nil {
			log.Println(err)
			http.Error(w, "Error writing response", http.StatusInternalServerError)
		}
	}
}

// meminfoParse reads and parses memory information from /proc/meminfo.
func meminfoParse() (map[string]int, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			panic(err)
		}
	}(file)

	// Parse the file line by line, and store the values in a map.
	memInfo := make(map[string]int)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		value, err := strconv.Atoi(strings.TrimSuffix(parts[1], " kB"))
		if err != nil {
			continue
		}

		// Convert from kB to MB.
		memInfo[parts[0][:len(parts[0])-1]] = value / 1024
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return memInfo, nil
}

// +----------------------------------------------------------------------------------------------+
// |                                           Database                                           |
// +----------------------------------------------------------------------------------------------+

func (s *Server) handleDevDatabaseRecreateFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := s.database("sqlite3.db")
		if err != nil {
			return
		}
	}
}

func (s *Server) handleDevDatabaseReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		schema := chi.URLParam(r, "schema")
		var purpose int
		if schema == "production" {
			purpose = db.PRODUCTION
		} else if schema == "simulation" {
			purpose = db.SIMULATION
		} else if schema == "simulation-full" {
			purpose = db.SIMULATION_FULL
		} else {
			http.Error(w, "Invalid schema", http.StatusBadRequest)
			return
		}

		initialUserCount, err := strconv.Atoi(chi.URLParam(r, "users"))
		if err != nil || initialUserCount < 3 || initialUserCount > 1_000_000 {
			http.Error(w, "Invalid number of users", http.StatusBadRequest)
			return
		}

		err = db.CreateSchema(conn, purpose, initialUserCount)
		if err != nil {
			http.Error(w, "Error creating schema", http.StatusInternalServerError)
			return
		}

		_, err = w.Write([]byte(fmt.Sprintf("Created %s schema with %d users", schema, initialUserCount)))
		if err != nil {
			log.Println(err)
			http.Error(w, "Error writing response", http.StatusInternalServerError)
		}
	}
}

func (s *Server) handleDevDatabaseGetFullDatabase() http.HandlerFunc {
	const query = `
		SELECT json_object(
			'users', (
				SELECT json_group_array(json_object(
					'uuid', uuid,
					'email', email,
					'password', password,
					'publicKey', public_key,
					'hasVoted', has_voted,
					'constituency', constituency,
					'firstName', first_name,
					'lastName', last_name,
					'isCentralAuthority', is_central_authority,
					'privateKey', private_key
				)) FROM users
			),
			'isEndOfElection', (
				SELECT json_group_array(json_object(
					'isEndOfElection', is_end_of_election
				)) FROM is_end_of_election
			)
		) as result;`

	return func(w http.ResponseWriter, r *http.Request) {
		conn := s.Database.Get(r.Context())
		defer s.Database.Put(conn)

		result, err := sqlitex.ResultText(conn.Prep(query))
		if err != nil {
			log.Println(err)
			http.Error(w, "Error getting the entire database", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(result))
		if err != nil {
			log.Println(err)
			http.Error(w, "Error writing the entire database", http.StatusInternalServerError)
		}
	}
}

// +----------------------------------------------------------------------------------------------+
// |                                          Blockchain                                          |
// +----------------------------------------------------------------------------------------------+

// handleDevBlockchainReset resets the blockchain by recreating it.
func (s *Server) handleDevBlockchainReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := os.Chdir("../blockchain"); err != nil {
			http.Error(w, "Error changing directory", http.StatusInternalServerError)
		} else if err := exec.Command("fablo", "prune").Run(); err != nil {
			http.Error(w, "Error destroying the blockchain", http.StatusInternalServerError)
		} else if err := exec.Command("fablo", "generate").Run(); err != nil {
			http.Error(w, "Error generating the blockchain configuration", http.StatusInternalServerError)
		} else if err := exec.Command("fablo", "up").Run(); err != nil {
			http.Error(w, "Error recreating the blockchain", http.StatusInternalServerError)
		} else if err := os.Chdir("../backend"); err != nil {
			http.Error(w, "Error changing directory", http.StatusInternalServerError)
		} else {
			respondPlainText(&w, "Successfully recreated the blockchain.")
		}
	}
}
