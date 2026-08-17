package database

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct{}

const File = "storage/database.db"

func (d *Database) Connect() *sql.DB {
	if configured := os.Getenv("AKOFLOW_DATABASE_PATH"); configured != "" {
		createDirectoryIfNotExists(filepath.Dir(configured))
		db, err := openSQLite(configured)
		if err != nil {
			panic(err)
		}
		return db
	}
	projectPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dbPath := filepath.Join(projectPath, "..", "..", File)

	createDirectoryIfNotExists(filepath.Dir(dbPath))

	db, err := openSQLite(dbPath)
	if err != nil {
		panic(err)
	}

	return db
}

func openSQLite(path string) (*sql.DB, error) {
	location := (&url.URL{Scheme: "file", Path: path}).String()
	dsn := location + "?_busy_timeout=10000&_journal_mode=WAL&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	return db, nil
}

func createDirectoryIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			println("Error creating directory", err.Error())
		}
	}

}
