package db

import (
	"testing"
)

func TestSchemaCorrectness(t *testing.T) {
	db := SetupTestDB(t)

	tables := []string{"users", "todos", "refresh_tokens"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %q not found: %v", table, err)
		}
	}
}
