package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed migrations.sql
var migrationsSQL string

func RunMigrations(db *sql.DB) error {
	statements := splitStatements(migrationsSQL)
	for i, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	var stmts []string
	for _, s := range strings.Split(sql, ";") {
		s = strings.TrimSpace(s)
		if s != "" {
			stmts = append(stmts, s)
		}
	}
	return stmts
}
