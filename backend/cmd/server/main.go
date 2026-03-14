package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"free-ship-checker-go/internal/httpapi"
)

// loadDotenv loads environment variables from a dotenv-formatted file at the given path.
// It parses lines of the form `KEY=VALUE`, ignoring empty lines and lines starting with `#`,
// trims whitespace, and sets each environment variable only if it is not already present.
// If the file cannot be opened the function returns without error; if an I/O error occurs
// while scanning the file a warning is logged.
func loadDotenv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
	if err := sc.Err(); err != nil {
		log.Printf("warning: error reading %s: %v", path, err)
	}
}

// main is the program entry point. It loads environment variables from ".env",
// determines the listen address (defaults to ":8085" when ADDR is unset),
// constructs the HTTP router and server (ReadHeaderTimeout = 10s), logs the
// listening address, and starts serving; the process exits on a non-closed
// server error.
func main() {
	loadDotenv(".env")
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8085"
	}

	mux := httpapi.NewRouter()
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
