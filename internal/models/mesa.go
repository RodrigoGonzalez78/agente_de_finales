package models

// Mesa representa una mesa de examen
type Mesa struct {
	ID           int    `json:"id"`
	Materia      string `json:"materia" form:"materia"`
	Turno        string `json:"turno" form:"turno"` // Ej: "1° Turno"
	Fecha        string `json:"fecha" form:"fecha"` // Ej: "20/02/2025"
	Hora         string `json:"hora" form:"hora"`
	Aula         string `json:"aula" form:"aula"`
	Sede         string `json:"sede"` // Populated via join
	Carrera      string `json:"carrera" form:"carrera"`
	FechaEdicion string `json:"fecha_edicion"`
}
