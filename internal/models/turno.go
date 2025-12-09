package models

type TurnoConfig struct {
	ID          int    `json:"id"`
	Nombre      string `json:"nombre" form:"nombre"`             // e.g. "1", "Julio"
	FechaInicio string `json:"fecha_inicio" form:"fecha_inicio"` // YYYY-MM-DD
	FechaFin    string `json:"fecha_fin" form:"fecha_fin"`       // YYYY-MM-DD
	Receso      bool   `json:"receso" form:"receso"`             // True if recess
}
