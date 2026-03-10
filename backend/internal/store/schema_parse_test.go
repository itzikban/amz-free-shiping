package store

import (
	"os"
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

func TestInitialSchemaParsesAsSQL(t *testing.T) {
	b, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pg_query.Parse(string(b)); err != nil {
		t.Fatalf("migration SQL should parse: %v", err)
	}
}
