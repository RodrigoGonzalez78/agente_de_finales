package repository

import (
	"database/sql"
	"fmt"
	"mi-bot-unne/internal/models"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ParamsRepository struct {
	DB *sql.DB
}

func NewParamsRepository(db *sql.DB) *ParamsRepository {
	repo := &ParamsRepository{DB: db}

	// Create table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS turnos_config (
			id INTEGER PRIMARY KEY, 
			nombre TEXT, 
			fecha_inicio TEXT, 
			fecha_fin TEXT, 
			receso INTEGER
		);
	`)
	if err != nil {
		fmt.Println("Error creating table turnos_config:", err)
	}

	repo.EnsureAulaUndefined()
	repo.SeedTurnos()

	return repo
}

func (r *ParamsRepository) GetAllSedes() ([]models.Sede, error) {
	rows, err := r.DB.Query("SELECT id, nombre FROM sedes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sedes []models.Sede
	for rows.Next() {
		var s models.Sede
		if err := rows.Scan(&s.ID, &s.Nombre); err != nil {
			return nil, err
		}
		sedes = append(sedes, s)
	}
	return sedes, nil
}

func (r *ParamsRepository) GetAulasBySede(sedeID int) ([]models.Aula, error) {
	rows, err := r.DB.Query("SELECT id, nombre, sede_id FROM aulas WHERE sede_id = ?", sedeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aulas []models.Aula
	for rows.Next() {
		var a models.Aula
		if err := rows.Scan(&a.ID, &a.Nombre, &a.SedeID); err != nil {
			return nil, err
		}
		aulas = append(aulas, a)
	}
	return aulas, nil
}

func (r *ParamsRepository) GetAllAulas() ([]models.Aula, error) {
	rows, err := r.DB.Query("SELECT id, nombre, sede_id FROM aulas")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aulas []models.Aula
	for rows.Next() {
		var a models.Aula
		if err := rows.Scan(&a.ID, &a.Nombre, &a.SedeID); err != nil {
			return nil, err
		}
		aulas = append(aulas, a)
	}
	return aulas, nil
}

func (r *ParamsRepository) GetAllCarreras() ([]models.Carrera, error) {
	rows, err := r.DB.Query("SELECT id, nombre FROM carreras")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var carreras []models.Carrera
	for rows.Next() {
		var c models.Carrera
		if err := rows.Scan(&c.ID, &c.Nombre); err != nil {
			return nil, err
		}
		carreras = append(carreras, c)
	}
	return carreras, nil
}

func (r *ParamsRepository) GetAllMaterias() ([]models.Materia, error) {
	rows, err := r.DB.Query("SELECT id, nombre FROM materias")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materias []models.Materia
	for rows.Next() {
		var m models.Materia
		if err := rows.Scan(&m.ID, &m.Nombre); err != nil {
			return nil, err
		}
		materias = append(materias, m)
	}
	return materias, nil
}

func (r *ParamsRepository) CreateMateria(nombre string) error {
	_, err := r.DB.Exec("INSERT INTO materias (nombre) VALUES (?)", nombre)
	return err
}

func (r *ParamsRepository) CreateSede(nombre string) error {
	_, err := r.DB.Exec("INSERT INTO sedes (nombre) VALUES (?)", nombre)
	return err
}

func (r *ParamsRepository) CreateAula(nombre string, sedeID int) error {
	_, err := r.DB.Exec("INSERT INTO aulas (nombre, sede_id) VALUES (?, ?)", nombre, sedeID)
	return err
}

func (r *ParamsRepository) CreateCarrera(nombre string) error {
	_, err := r.DB.Exec("INSERT INTO carreras (nombre) VALUES (?)", nombre)
	return err
}

func (r *ParamsRepository) EnsureAulaUndefined() {
	var count int
	r.DB.QueryRow("SELECT COUNT(*) FROM aulas WHERE nombre = 'Sin definir'").Scan(&count)
	if count == 0 {
		r.DB.Exec("INSERT INTO aulas (nombre, sede_id) VALUES ('Sin definir', 0)")
	}
}

func (r *ParamsRepository) SeedTurnos() {
	var count int
	r.DB.QueryRow("SELECT COUNT(*) FROM turnos_config").Scan(&count)
	if count == 0 {
		// Specific Seeding data provided by User
		// 1° Turno 17 al 21 de febrero (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "1° Turno", "2025-02-17", "2025-02-21", 1)
		// 2° Turno 10 al 14 de marzo (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "2° Turno", "2025-03-10", "2025-03-14", 1)
		// 3° Turno 25 al 31 de marzo (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "3° Turno", "2025-03-25", "2025-03-31", 1)
		// 4° Turno 05 al 30 de mayo (Receso=false - Mesa expandida)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "4° Turno", "2025-05-05", "2025-05-30", 0)
		// 5° Turno 30 junio al 4 de julio (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "5° Turno", "2025-06-30", "2025-07-04", 1)
		// 6° Turno 28 de julio al 01 de agosto (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "6° Turno", "2025-07-28", "2025-08-01", 1)
		// 7° Turno 01 al 26 de septiembre (Receso=false - Mesa expandida)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "7° Turno", "2025-09-01", "2025-09-26", 0)
		// 8° Turno 06 al 31 de octubre (Receso=false)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "8° Turno", "2025-10-06", "2025-10-31", 0)
		// 9° Turno 24 de noviembre al 1 de diciembre (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "9° Turno", "2025-11-24", "2025-12-01", 1)
		// 10° Turno 15 al 19 de diciembre (Receso=true)
		r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", "10° Turno", "2025-12-15", "2025-12-19", 1)
	}
}

func (r *ParamsRepository) CreateTurnoConfig(t models.TurnoConfig) error {
	recesoInt := 0
	if t.Receso {
		recesoInt = 1
	}
	_, err := r.DB.Exec("INSERT INTO turnos_config (nombre, fecha_inicio, fecha_fin, receso) VALUES (?, ?, ?, ?)", t.Nombre, t.FechaInicio, t.FechaFin, recesoInt)
	return err
}

func (r *ParamsRepository) UpdateTurnoConfig(t models.TurnoConfig) error {
	recesoInt := 0
	if t.Receso {
		recesoInt = 1
	}
	_, err := r.DB.Exec("UPDATE turnos_config SET nombre=?, fecha_inicio=?, fecha_fin=?, receso=? WHERE id=?", t.Nombre, t.FechaInicio, t.FechaFin, recesoInt, t.ID)
	return err
}

func (r *ParamsRepository) GetTurnoConfigs() ([]models.TurnoConfig, error) {
	rows, err := r.DB.Query("SELECT id, nombre, fecha_inicio, fecha_fin, receso FROM turnos_config ORDER BY CAST(nombre AS INTEGER) ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turnos []models.TurnoConfig
	for rows.Next() {
		var t models.TurnoConfig
		var recesoInt int
		if err := rows.Scan(&t.ID, &t.Nombre, &t.FechaInicio, &t.FechaFin, &recesoInt); err != nil {
			return nil, err
		}
		t.Receso = recesoInt == 1
		turnos = append(turnos, t)
	}
	return turnos, nil
}

func (r *ParamsRepository) DeleteTurnoConfig(id int) error {
	_, err := r.DB.Exec("DELETE FROM turnos_config WHERE id = ?", id)
	return err
}

// --- CRUD Operations ---

// Materia
func (r *ParamsRepository) GetMateria(id int) (models.Materia, error) {
	var m models.Materia
	err := r.DB.QueryRow("SELECT id, nombre FROM materias WHERE id = ?", id).Scan(&m.ID, &m.Nombre)
	return m, err
}
func (r *ParamsRepository) UpdateMateria(id int, nombre string) error {
	_, err := r.DB.Exec("UPDATE materias SET nombre = ? WHERE id = ?", nombre, id)
	return err
}
func (r *ParamsRepository) DeleteMateria(id int) error {
	_, err := r.DB.Exec("DELETE FROM materias WHERE id = ?", id)
	return err
}

// Carrera
func (r *ParamsRepository) GetCarrera(id int) (models.Carrera, error) {
	var c models.Carrera
	err := r.DB.QueryRow("SELECT id, nombre FROM carreras WHERE id = ?", id).Scan(&c.ID, &c.Nombre)
	return c, err
}
func (r *ParamsRepository) UpdateCarrera(id int, nombre string) error {
	_, err := r.DB.Exec("UPDATE carreras SET nombre = ? WHERE id = ?", nombre, id)
	return err
}
func (r *ParamsRepository) DeleteCarrera(id int) error {
	_, err := r.DB.Exec("DELETE FROM carreras WHERE id = ?", id)
	return err
}

// Sede
func (r *ParamsRepository) GetSede(id int) (models.Sede, error) {
	var s models.Sede
	err := r.DB.QueryRow("SELECT id, nombre FROM sedes WHERE id = ?", id).Scan(&s.ID, &s.Nombre)
	return s, err
}
func (r *ParamsRepository) UpdateSede(id int, nombre string) error {
	_, err := r.DB.Exec("UPDATE sedes SET nombre = ? WHERE id = ?", nombre, id)
	return err
}
func (r *ParamsRepository) DeleteSede(id int) error {
	_, err := r.DB.Exec("DELETE FROM sedes WHERE id = ?", id)
	return err
}

// Aula
func (r *ParamsRepository) GetAula(id int) (models.Aula, error) {
	var a models.Aula
	err := r.DB.QueryRow("SELECT id, nombre, sede_id FROM aulas WHERE id = ?", id).Scan(&a.ID, &a.Nombre, &a.SedeID)
	return a, err
}
func (r *ParamsRepository) UpdateAula(id int, nombre string, sedeID int) error {
	_, err := r.DB.Exec("UPDATE aulas SET nombre = ?, sede_id = ? WHERE id = ?", nombre, sedeID, id)
	return err
}
func (r *ParamsRepository) DeleteAula(id int) error {
	_, err := r.DB.Exec("DELETE FROM aulas WHERE id = ?", id)
	return err
}

// GetFutureTurnos returns turnos with fecha_inicio >= today
func (r *ParamsRepository) GetFutureTurnos() ([]models.TurnoConfig, error) {
	rows, err := r.DB.Query(`
SELECT id, nombre, fecha_inicio, fecha_fin, receso 
FROM turnos_config 
ORDER BY fecha_inicio ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turnos []models.TurnoConfig
	today := time.Now()

	for rows.Next() {
		var t models.TurnoConfig
		var recesoInt int
		if err := rows.Scan(&t.ID, &t.Nombre, &t.FechaInicio, &t.FechaFin, &recesoInt); err != nil {
			return nil, err
		}
		t.Receso = recesoInt == 1

		// Parse fecha_inicio (YYYY-MM-DD)
		if fechaParsed, err := time.Parse("2006-01-02", t.FechaInicio); err == nil {
			if fechaParsed.After(today) || fechaParsed.Equal(today.Truncate(24*time.Hour)) {
				turnos = append(turnos, t)
			}
		}
	}
	return turnos, nil
}
