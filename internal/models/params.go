package models

type Sede struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}

type Aula struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
	SedeID int    `json:"sede_id"`
}

type Carrera struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}

type Materia struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}
