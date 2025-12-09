package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"mi-bot-unne/internal/models"
	"mi-bot-unne/internal/repository"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	Repo       *repository.MesaRepository
	ParamsRepo *repository.ParamsRepository
}

func NewAdminHandler(repo *repository.MesaRepository, paramsRepo *repository.ParamsRepository) *AdminHandler {
	return &AdminHandler{Repo: repo, ParamsRepo: paramsRepo}
}

func (h *AdminHandler) ShowDashboard(c *gin.Context) {
	mesas, err := h.Repo.GetAll()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error leyendo DB")
		return
	}

	// Fetch existing data for drop downs
	sedes, _ := h.ParamsRepo.GetAllSedes()
	carreras, _ := h.ParamsRepo.GetAllCarreras()
	materias, _ := h.ParamsRepo.GetAllMaterias()
	turnos, _ := h.ParamsRepo.GetTurnoConfigs()

	c.HTML(http.StatusOK, "admin.html", gin.H{
		"mesas":    mesas,
		"sedes":    sedes,
		"carreras": carreras,
		"materias": materias,
		"turnos":   turnos, // Now available in dashboard
	})
}

func (h *AdminHandler) GetAulas(c *gin.Context) {
	sedeIDStr := c.Query("sede_id")
	sedeID, err := strconv.Atoi(sedeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sede_id"})
		return
	}

	aulas, err := h.ParamsRepo.GetAulasBySede(sedeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching aulas"})
		return
	}
	c.JSON(http.StatusOK, aulas)
}

func (h *AdminHandler) ShowParams(c *gin.Context) {
	sedes, err := h.ParamsRepo.GetAllSedes()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error leyendo Sedes")
		return
	}

	carreras, err := h.ParamsRepo.GetAllCarreras()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error leyendo Carreras")
		return
	}

	materias, err := h.ParamsRepo.GetAllMaterias()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error leyendo Materias")
		return
	}

	aulas, err := h.ParamsRepo.GetAllAulas()
	if err != nil {
		c.String(http.StatusInternalServerError, "Error leyendo Aulas")
		return
	}
	turnos, _ := h.ParamsRepo.GetTurnoConfigs()

	c.HTML(http.StatusOK, "admin_params.html", gin.H{
		"sedes":    sedes,
		"carreras": carreras,
		"materias": materias,
		"aulas":    aulas,
		"turnos":   turnos,
	})
}

func (h *AdminHandler) StoreCarrera(c *gin.Context) {
	nombre := c.PostForm("nombre")
	if nombre != "" {
		h.ParamsRepo.CreateCarrera(nombre)
	}
	c.Redirect(http.StatusFound, "/admin/config")
}

func (h *AdminHandler) StoreMateria(c *gin.Context) {
	nombre := c.PostForm("nombre")
	if nombre != "" {
		h.ParamsRepo.CreateMateria(nombre)
	}
	c.Redirect(http.StatusFound, "/admin/config")
}

func (h *AdminHandler) StoreSede(c *gin.Context) {
	nombre := c.PostForm("nombre")
	if nombre != "" {
		h.ParamsRepo.CreateSede(nombre)
	}
	c.Redirect(http.StatusFound, "/admin/config")
}

func (h *AdminHandler) StoreAula(c *gin.Context) {
	nombre := c.PostForm("nombre")
	sedeID, _ := strconv.Atoi(c.PostForm("sede_id"))
	if err := h.ParamsRepo.CreateAula(nombre, sedeID); err != nil {
		c.String(http.StatusInternalServerError, "Error al crear aula")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/config")
}

func (h *AdminHandler) StoreTurnoConfig(c *gin.Context) {
	var t models.TurnoConfig
	// Checkbox handling: "on" or absent. Gin Bind might struggle with unchecked = false if using ShouldBind.
	// We'll rely on PostForm for boolean safety if ShouldBind fails on unchecked.
	t.Nombre = c.PostForm("nombre")
	t.FechaInicio = c.PostForm("fecha_inicio")
	t.FechaFin = c.PostForm("fecha_fin")
	if c.PostForm("receso") == "on" {
		t.Receso = true
	} else {
		t.Receso = false
	}

	if err := h.ParamsRepo.CreateTurnoConfig(t); err != nil {
		c.String(http.StatusInternalServerError, "Error al crear turno")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/config")
}

func (h *AdminHandler) UpdateTurnoConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id")) // From URL param if used directly or hidden input
	// Actually for "quick edit" we might post to /update/:id

	var t models.TurnoConfig
	t.ID = id
	t.Nombre = c.PostForm("nombre")
	t.FechaInicio = c.PostForm("fecha_inicio")
	t.FechaFin = c.PostForm("fecha_fin")
	if c.PostForm("receso") == "on" {
		t.Receso = true
	} else {
		t.Receso = false
	}

	if err := h.ParamsRepo.UpdateTurnoConfig(t); err != nil {
		c.String(http.StatusInternalServerError, "Error al actualizar turno")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/config")
}

func (h *AdminHandler) DeleteTurnoConfig(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.ParamsRepo.DeleteTurnoConfig(id); err != nil {
		c.String(http.StatusInternalServerError, "Error al eliminar turno")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/config")
}

func (h *AdminHandler) CreateMesa(c *gin.Context) {
	// Debugging: Print raw form values
	log.Printf("DEBUG FORM: Carrera=%s, Hora=%s, Aula=%s, SedeID=%s",
		c.PostForm("carrera"), c.PostForm("hora"), c.PostForm("aula"), c.PostForm("sede"))

	var nuevaMesa models.Mesa
	if err := c.ShouldBind(&nuevaMesa); err != nil {
		log.Printf("BIND ERROR: %v", err)
		c.String(http.StatusBadRequest, "Datos inválidos")
		return
	}

	// Set current timestamp
	nuevaMesa.FechaEdicion = time.Now().Format("2006-01-02 15:04:05")

	log.Printf("STRUCT AFTER BIND: %+v", nuevaMesa)

	if err := h.Repo.Create(nuevaMesa); err != nil {
		log.Printf("DB ERROR: %v", err)
		c.String(http.StatusInternalServerError, "Error guardando en DB")
		return
	}

	c.Redirect(http.StatusFound, "/admin")
}

func (h *AdminHandler) DeleteMesa(c *gin.Context) {
	id := c.Param("id")
	if err := h.Repo.Delete(id); err != nil {
		c.String(http.StatusInternalServerError, "Error eliminando mesa")
		return
	}
	c.Redirect(http.StatusFound, "/admin")
}

// --- Generic / Specific Param Management ---

func (h *AdminHandler) ShowEditParam(c *gin.Context) {
	paramType := c.Param("type")
	id, _ := strconv.Atoi(c.Param("id"))

	var data gin.H
	var err error

	switch paramType {
	case "materia":
		var m models.Materia
		m, err = h.ParamsRepo.GetMateria(id)
		data = gin.H{"type": "materia", "id": m.ID, "nombre": m.Nombre}
	case "carrera":
		var ca models.Carrera
		ca, err = h.ParamsRepo.GetCarrera(id)
		data = gin.H{"type": "carrera", "id": ca.ID, "nombre": ca.Nombre}
	case "sede":
		var s models.Sede
		s, err = h.ParamsRepo.GetSede(id)
		data = gin.H{"type": "sede", "id": s.ID, "nombre": s.Nombre}
	case "aula":
		var a models.Aula
		a, err = h.ParamsRepo.GetAula(id)
		sedes, _ := h.ParamsRepo.GetAllSedes()
		data = gin.H{"type": "aula", "id": a.ID, "nombre": a.Nombre, "sede_id": a.SedeID, "sedes": sedes}
	default:
		c.String(http.StatusBadRequest, "Tipo inválido")
		return
	}

	if err != nil {
		c.String(http.StatusNotFound, "Elemento no encontrado")
		return
	}

	c.HTML(http.StatusOK, "admin_edit_param.html", data)
}

func (h *AdminHandler) UpdateParam(c *gin.Context) {
	paramType := c.Param("type")
	id, _ := strconv.Atoi(c.Param("id"))
	nombre := c.PostForm("nombre")

	var err error
	switch paramType {
	case "materia":
		err = h.ParamsRepo.UpdateMateria(id, nombre)
	case "carrera":
		err = h.ParamsRepo.UpdateCarrera(id, nombre)
	case "sede":
		err = h.ParamsRepo.UpdateSede(id, nombre)
	case "aula":
		sedeID, _ := strconv.Atoi(c.PostForm("sede_id"))
		err = h.ParamsRepo.UpdateAula(id, nombre, sedeID)
	default:
		c.String(http.StatusBadRequest, "Tipo inválido")
		return
	}

	if err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando")
		return
	}
	c.Redirect(http.StatusFound, "/admin/config")
}

func (h *AdminHandler) DeleteParam(c *gin.Context) {
	paramType := c.Param("type")
	id, _ := strconv.Atoi(c.Param("id"))

	var err error
	switch paramType {
	case "materia":
		err = h.ParamsRepo.DeleteMateria(id)
	case "carrera":
		err = h.ParamsRepo.DeleteCarrera(id)
	case "sede":
		err = h.ParamsRepo.DeleteSede(id)
	case "aula":
		err = h.ParamsRepo.DeleteAula(id)
	default:
		c.String(http.StatusBadRequest, "Tipo inválido")
		return
	}

	if err != nil {
		c.String(http.StatusInternalServerError, "Error eliminando")
		return
	}
	c.Redirect(http.StatusFound, "/admin/config")
}
