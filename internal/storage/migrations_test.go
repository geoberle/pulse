package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrations_SchemaGolden(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Query("SELECT type, name, sql FROM sqlite_master WHERE sql IS NOT NULL ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()

	var lines []string
	for rows.Next() {
		var typ, name, ddl string
		if err := rows.Scan(&typ, &name, &ddl); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, ddl+";")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	sort.Strings(lines)
	got := strings.Join(lines, "\n") + "\n"

	goldenPath := filepath.Join("testdata", "schema.golden")

	if os.Getenv("UPDATE") == "1" {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file not found; run with UPDATE=1: %v", err)
	}
	if got != string(want) {
		t.Errorf("schema mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}

func TestMigrations_Idempotent(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	var version int
	if err := db.QueryRow("SELECT MAX(version) FROM migrations").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != len(migrations) {
		t.Errorf("expected version %d, got %d", len(migrations), version)
	}
}
