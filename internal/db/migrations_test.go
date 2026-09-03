package db

import (
	"testing"
)

func TestMigrationsIdempotent(t *testing.T) {
	db := SetupTestDB(t)

	if err := RunMigrations(db); err != nil {
		t.Errorf("First RunMigrations failed: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Errorf("Second RunMigrations failed (should be idempotent): %v", err)
	}
}
