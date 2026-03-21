package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"free-ship-checker-go/internal/altcache"
	"free-ship-checker-go/internal/httpapi"
	"free-ship-checker-go/internal/store"
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
// initializes optional DB pool and Redis client for caching,
// constructs the HTTP router and server (ReadHeaderTimeout = 10s), logs the
// listening address, and starts serving; the process exits on a non-closed
// server error.
func main() {
	loadDotenv(".env")
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8085"
	}

	// Initialize DB pool (optional — graceful degradation if missing)
	var dbPool *pgxpool.Pool
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		if pool, err := store.Connect(context.Background()); err == nil {
			dbPool = pool
			defer dbPool.Close()
			log.Printf("Connected to database")
		} else {
			log.Printf("warning: database connection failed, altcache DB layer disabled: %v", err)
		}
	} else {
		log.Printf("warning: DATABASE_URL not set, altcache DB layer disabled")
	}

	// Initialize Redis client (optional — graceful degradation if missing)
	var rdb *redis.Client
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		rdb = redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: os.Getenv("REDIS_PASSWORD"),
		})
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			log.Printf("warning: Redis connection failed, altcache Redis layer disabled: %v", err)
			rdb = nil
		} else {
			defer rdb.Close()
			log.Printf("Connected to Redis at %s", redisAddr)
		}
	} else {
		log.Printf("warning: REDIS_ADDR not set, altcache Redis layer disabled")
	}

	// Initialize alternative cache (if any layer is available)
	var altCache *altcache.Cache
	if rdb != nil || dbPool != nil {
		altCache = altcache.New(rdb, dbPool)
		log.Printf("Alternative cache initialized (Redis: %v, DB: %v)", rdb != nil, dbPool != nil)
	} else {
		log.Printf("warning: no cache layers available, PA-API alternatives will not be cached")
	}

	mux := httpapi.NewRouter(altCache)
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
