package repository

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	file string
}

const File = "storage/database.db"

func (d *Database) Connect() *sql.DB {
	projectPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dbPath := filepath.Join(projectPath, "..", "..", File)

	createDirectoryIfNotExists(filepath.Dir(dbPath))

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}

	return db
}

func createDirectoryIfNotExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			println("Error creating directory", err.Error())
		}
	}

}
