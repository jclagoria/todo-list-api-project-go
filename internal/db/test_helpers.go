package db

import (
	"database/sql"
	"testing"
)

func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := ConnectTest()
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
