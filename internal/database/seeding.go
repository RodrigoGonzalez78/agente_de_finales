package database

import "database/sql"

func seedData(db *sql.DB) {
	// Check if data exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sedes").Scan(&count)
	if count > 0 {
		return
	}

	// Seed Sedes
	sedes := []string{"Campus Resistencia", "Campus Corrientes", "Edificio Central"}
	for _, s := range sedes {
		db.Exec("INSERT INTO sedes (nombre) VALUES (?)", s)
	}

	// Seed Carreras
	carreras := []string{"Ingeniería en Sistemas", "Licenciatura en Matemática", "Profesorado en Física"}
	for _, c := range carreras {
		db.Exec("INSERT INTO carreras (nombre) VALUES (?)", c)
	}

	// Seed Aulas
	// ID 1: Campus Resistencia
	db.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", "Aula 1 - PB", 1)
	db.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", "Aula 2 - PB", 1)
	db.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", "Aula Magna", 1)
	// ID 2: Campus Corrientes
	db.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", "Laboratorio 1", 2)
	db.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", "Laboratorio 2", 2)
	// ID 3: Edificio Central
	db.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", "Sala de Conferencias", 3)

	// Seed Materias
	materias := []string{"Álgebra I", "Análisis Matemático I", "Física I", "Algoritmos y Estructuras de Datos", "Sistemas Operativos"}
	for _, m := range materias {
		db.Exec("INSERT INTO materias (nombre) VALUES (?)", m)
	}
}
