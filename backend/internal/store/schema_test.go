package store

import (
	"os"
	"strings"
	"testing"
)

func TestInitialSchemaContainsCoreTables(t *testing.T) {
	b, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	s := strings.ToLower(string(b))

	mustContain := []string{
		"create table if not exists users",
		"create table if not exists tracked_items",
		"create table if not exists snapshots",
		"create table if not exists alerts",
		"create table if not exists outbox",
		"create table if not exists notification_attempts",
	}

	for _, m := range mustContain {
		if !strings.Contains(s, m) {
			t.Fatalf("schema missing: %s", m)
		}
	}
}
