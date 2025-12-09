package repository

import (
	"database/sql"
	"mi-bot-unne/internal/models"
	"strings"
	"time"
)

type MesaRepository struct {
	DB *sql.DB
}

func NewMesaRepository(db *sql.DB) *MesaRepository {
	// Simple migration: try to add the column, ignore error if it exists
	// In a real app, use a migration tool or check schema first.
	// SQLite 'ADD COLUMN' is safe if we ignore duplication errors or check PRAGMA table_info.
	// For simplicity in this agentic context, we'll brute-force try to add it.
	_, _ = db.Exec("ALTER TABLE mesas ADD COLUMN fecha_edicion TEXT")
	return &MesaRepository{DB: db}
}

func (r *MesaRepository) GetAll() ([]models.Mesa, error) {
	rows, err := r.DB.Query("SELECT id, materia, turno, fecha, hora, aula, carrera, COALESCE(fecha_edicion, '') FROM mesas ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mesas []models.Mesa
	for rows.Next() {
		var m models.Mesa
		if err := rows.Scan(&m.ID, &m.Materia, &m.Turno, &m.Fecha, &m.Hora, &m.Aula, &m.Carrera, &m.FechaEdicion); err != nil {
			return nil, err
		}
		mesas = append(mesas, m)
	}
	return mesas, nil
}

func (r *MesaRepository) Create(m models.Mesa) error {
	stmt, err := r.DB.Prepare("INSERT INTO mesas(materia, turno, fecha, hora, aula, carrera, fecha_edicion) VALUES(?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(m.Materia, m.Turno, m.Fecha, m.Hora, m.Aula, m.Carrera, m.FechaEdicion)
	return err
}

func (r *MesaRepository) Delete(id string) error {
	stmt, err := r.DB.Prepare("DELETE FROM mesas WHERE id = ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id)
	return err
}

func (r *MesaRepository) SearchWithFilter(materia, mesaFilter string) ([]models.Mesa, error) {
	// Query matches Mesa/Turno first (DB side)
	sqlQuery := `
		SELECT 
			m.materia, m.turno, m.fecha, m.aula, m.hora, m.carrera, 
			COALESCE(m.fecha_edicion, ''), COALESCE(s.nombre, '')
		FROM mesas m
		LEFT JOIN aulas a ON m.aula = a.nombre 
		LEFT JOIN sedes s ON a.sede_id = s.id
		WHERE (CAST(m.id AS TEXT) = ? OR m.turno LIKE ?)
	`

	rows, err := r.DB.Query(sqlQuery, mesaFilter, "%"+mesaFilter+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []models.Mesa
	normalizedInput := Normalize(materia)

	for rows.Next() {
		var m models.Mesa
		if err := rows.Scan(&m.Materia, &m.Turno, &m.Fecha, &m.Aula, &m.Hora, &m.Carrera, &m.FechaEdicion, &m.Sede); err != nil {
			return nil, err
		}
		if m.Sede == "" {
			m.Sede = "Sin asignar / Consultar"
		}

		// Go-side Fuzzy Matching
		if strings.Contains(Normalize(m.Materia), normalizedInput) {
			resultados = append(resultados, m)
		}
	}
	return resultados, nil
}

func (r *MesaRepository) GetUniqueMaterias(pattern string) ([]string, error) {
	// Fetch ALL distinct materias, then filter in Go
	rows, err := r.DB.Query("SELECT DISTINCT materia FROM mesas")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materias []string
	normalizedPattern := Normalize(pattern)

	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		if strings.Contains(Normalize(m), normalizedPattern) {
			materias = append(materias, m)
		}
	}
	return materias, nil
}

// Helper for loose matching (case-insensitive + ignore accents)
func Normalize(s string) string {
	s = strings.ToLower(s)
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	)
	return r.Replace(s)
}

func (r *MesaRepository) GetFullSchedule(materia string) ([]models.Mesa, error) {
	sqlQuery := `
		SELECT 
			m.materia, m.turno, m.fecha, m.aula, m.hora, m.carrera, 
			COALESCE(m.fecha_edicion, ''), COALESCE(s.nombre, '')
		FROM mesas m
		LEFT JOIN aulas a ON m.aula = a.nombre 
		LEFT JOIN sedes s ON a.sede_id = s.id
		WHERE m.materia = ?
		ORDER BY m.id ASC
	`

	rows, err := r.DB.Query(sqlQuery, materia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []models.Mesa
	for rows.Next() {
		var m models.Mesa
		if err := rows.Scan(&m.Materia, &m.Turno, &m.Fecha, &m.Aula, &m.Hora, &m.Carrera, &m.FechaEdicion, &m.Sede); err != nil {
			return nil, err
		}
		if m.Sede == "" {
			m.Sede = "Sin asignar"
		}
		resultados = append(resultados, m)
	}
	return resultados, nil
}

// GetFutureDates returns only mesas with dates >= today
func (r *MesaRepository) GetFutureDates(materia string) ([]models.Mesa, error) {
	sqlQuery := `
		SELECT 
			m.materia, m.turno, m.fecha, m.aula, m.hora, m.carrera, 
			COALESCE(m.fecha_edicion, ''), COALESCE(s.nombre, '')
		FROM mesas m
		LEFT JOIN aulas a ON m.aula = a.nombre 
		LEFT JOIN sedes s ON a.sede_id = s.id
		WHERE m.materia = ?
		ORDER BY m.id ASC
	`

	rows, err := r.DB.Query(sqlQuery, materia)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resultados []models.Mesa
	today := time.Now()

	for rows.Next() {
		var m models.Mesa
		if err := rows.Scan(&m.Materia, &m.Turno, &m.Fecha, &m.Aula, &m.Hora, &m.Carrera, &m.FechaEdicion, &m.Sede); err != nil {
			return nil, err
		}
		if m.Sede == "" {
			m.Sede = "Sin asignar"
		}

		// Parse date (DD/MM/YYYY format)
		if fechaParsed, err := time.Parse("02/01/2006", m.Fecha); err == nil {
			if fechaParsed.After(today) || fechaParsed.Equal(today.Truncate(24*time.Hour)) {
				resultados = append(resultados, m)
			}
		} else {
			// If parse fails, include it anyway
			resultados = append(resultados, m)
		}
	}
	return resultados, nil
}

// GetByTurn returns mesas for a specific turn/turno
func (r *MesaRepository) GetByTurn(materia string, turno string) (models.Mesa, error) {
	sqlQuery := `
		SELECT 
			m.materia, m.turno, m.fecha, m.aula, m.hora, m.carrera, 
			COALESCE(m.fecha_edicion, ''), COALESCE(s.nombre, '')
		FROM mesas m
		LEFT JOIN aulas a ON m.aula = a.nombre 
		LEFT JOIN sedes s ON a.sede_id = s.id
		WHERE m.materia = ? AND m.turno = ?
		LIMIT 1
	`

	var m models.Mesa
	err := r.DB.QueryRow(sqlQuery, materia, turno).Scan(&m.Materia, &m.Turno, &m.Fecha, &m.Aula, &m.Hora, &m.Carrera, &m.FechaEdicion, &m.Sede)
	if m.Sede == "" {
		m.Sede = "Sin asignar"
	}
	return m, err
}
