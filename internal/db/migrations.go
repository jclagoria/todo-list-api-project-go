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

// splitStatements splits SQL on semicolons, ignoring those inside single-quoted strings.
func splitStatements(sql string) []string {
	var stmts []string
	var buf strings.Builder
	inQuote := false

	for _, ch := range sql {
		switch {
		case ch == '\'' && !inQuote:
			inQuote = true
		case ch == '\'' && inQuote:
			inQuote = false
		case ch == ';' && !inQuote:
			s := strings.TrimSpace(buf.String())
			if s != "" {
				stmts = append(stmts, s)
			}
			buf.Reset()
			continue
		}
		buf.WriteRune(ch)
	}
	if s := strings.TrimSpace(buf.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}
