package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB inicializa la conexión a la base de datos y crea la tabla si no existe
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	// Crear tabla mesas si no existe
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS mesas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		materia TEXT,
		turno TEXT,
		fecha TEXT,
		hora TEXT,
		aula TEXT,
		carrera TEXT
	);
	CREATE TABLE IF NOT EXISTS sedes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT
	);
	CREATE TABLE IF NOT EXISTS aulas (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT,
		sede_id INTEGER,
		FOREIGN KEY(sede_id) REFERENCES sedes(id)
	);
	CREATE TABLE IF NOT EXISTS carreras (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT
	);
	CREATE TABLE IF NOT EXISTS materias (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return nil, err
	}

	// Seed data (Simple check to see if we need to seed)
	seedData(db)

	return db, nil
}
