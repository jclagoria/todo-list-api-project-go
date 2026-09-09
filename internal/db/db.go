package db

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib" // Register PostgreSQL driver for database/sql
	_ "github.com/mattn/go-sqlite3"    // Register SQLite driver for database/sql
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func ConnectTest() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	if err := RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
