package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mi-bot-unne/internal/models"
	"mi-bot-unne/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for dev
	},
}

type ChatHandler struct {
	Repo       *repository.MesaRepository
	ParamsRepo *repository.ParamsRepository
}

func NewChatHandler(repo *repository.MesaRepository, paramsRepo *repository.ParamsRepository) *ChatHandler {
	return &ChatHandler{
		Repo:       repo,
		ParamsRepo: paramsRepo,
	}
}

func (h *ChatHandler) ShowChat(c *gin.Context) {
	c.HTML(http.StatusOK, "chat.html", nil)
}

func (h *ChatHandler) HandleMessage(c *gin.Context) {
	mensajeUsuario := c.PostForm("mensaje")
	step := c.PostForm("step")
	mesaFilter := c.PostForm("mesa_filter")

	var responseHTML string
	var nextStep string
	var nextFilter string

	// State Machine
	switch step {
	case "":
		// Initial State. User sees "Enter Mesa" in UI and types it.
		// If message is generic "Hola", ask for Mesa.
		// If message looks like input, treat as Mesa and ask for Materia.
		if mensajeUsuario == "" || mensajeUsuario == "Hola" || mensajeUsuario == "hola" {
			responseHTML = `
		<div class="message-container bot">
			<div class="message-content">
				<div class="avatar bot">✨</div>
				<div class="text-content"><p>Hola, por favor ingresá el <strong>Número de Mesa</strong> o <strong>Turno</strong> para buscar.</p></div>
			</div>
		</div>`
			nextStep = "1"
		} else {
			// Assume input is Mesa ID
			nextFilter = mensajeUsuario
			responseHTML = `
		<div class="message-container user"><div class="message-content" style="justify-content: flex-end;"><div class="text-content" style="background-color: #2F3031; padding: 10px 16px; border-radius: 20px 2px 20px 20px; max-width: 70%;">` + mensajeUsuario + `</div></div></div>
		<div class="message-container bot">
			<div class="message-content">
				<div class="avatar bot">✨</div>
				<div class="text-content"><p>Dale, busco en Mesa/Turno <strong>` + mensajeUsuario + `</strong>. <br>Ahora decime, ¿qué <strong>Materia</strong> buscás?</p></div>
			</div>
		</div>`
			nextStep = "2"
		}

	case "1":
		// Received Mesa/Turno -> Ask for Materia
		// mesaFilter becomes the user's message (e.g., "1")
		nextFilter = mensajeUsuario
		responseHTML = `
		<div class="message-container user"><div class="message-content" style="justify-content: flex-end;"><div class="text-content" style="background-color: #2F3031; padding: 10px 16px; border-radius: 20px 2px 20px 20px; max-width: 70%;">` + mensajeUsuario + `</div></div></div>
		<div class="message-container bot">
			<div class="message-content">
				<div class="avatar bot">✨</div>
				<div class="text-content"><p>Perfecto. Ahora ingresá el nombre de la <strong>Materia</strong>.</p></div>
			</div>
		</div>`
		nextStep = "2"

	case "2":
		// Received Materia -> Perform Search with Filter
		resultados, err := h.Repo.SearchWithFilter(mensajeUsuario, mesaFilter)
		if err != nil {
			log.Println(err)
			c.String(http.StatusInternalServerError, "Error en el servidor")
			return
		}

		c.HTML(http.StatusOK, "respuesta_fragmento.html", gin.H{
			"mensaje_usuario": mensajeUsuario,
			"resultados":      resultados,
			"encontrado":      len(resultados) > 0,
			"reset_state":     true, // Flag to reset form
		})
		return
	}

	// For intermediate steps, we render the partial HTML directly + OOB swap
	// We append the OOB swap div
	oobSwap := `
	<div id="form-state" hx-swap-oob="true">
		<input type="hidden" name="step" value="` + nextStep + `">
		<input type="hidden" name="mesa_filter" value="` + nextFilter + `">
	</div>`

	c.Writer.WriteString(responseHTML + oobSwap)
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade:", err)
		return
	}
	defer conn.Close()

	// No need to send initial greeting - it's already in the HTML
	// The menu is pre-rendered for instant display

	// State machine variables
	var currentOption string // "1", "2", "3"
	var currentTurn string   // For option 2
	var awaitingMateria bool
	var awaitingTurn bool
	var awaitingDownload bool
	var pendingCardID string

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read Error:", err)
			break
		}
		mensajeUsuario := strings.ToLower(strings.TrimSpace(string(msg)))

		// PHASE 0: Check if waiting for download confirmation (HIGHEST PRIORITY)
		if awaitingDownload {
			if mensajeUsuario == "si" || mensajeUsuario == "sí" || mensajeUsuario == "s" {
				// Send download trigger using a self-clicking button
				downloadHTML := `<div style="display:none;"><button id="auto-download-btn" onclick="downloadCard('` + pendingCardID + `'); this.remove();">Download</button></div>`
				downloadHTML += `<script>setTimeout(() => { const btn = document.getElementById('auto-download-btn'); if(btn) btn.click(); }, 100);</script>`
				downloadHTML += botMsg("¡Perfecto! Descargando imagen... 📥")
				conn.WriteMessage(websocket.TextMessage, []byte(downloadHTML))
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Entendido 👍")))
			}
			awaitingDownload = false
			pendingCardID = ""
			sendMenuOptions(conn)
			continue // Skip to next iteration
		}

		// PHASE 1: Awaiting Turn Number (for Option 2)
		if awaitingTurn {
			turnNum, err := strconv.Atoi(mensajeUsuario)
			if err != nil || turnNum < 1 || turnNum > 10 {
				conn.WriteMessage(websocket.TextMessage, []byte(botMsg("⚠️ Por favor ingresá solo un número del 1 al 10.")))
				continue
			}
			currentTurn = strconv.Itoa(turnNum) + "° Turno"
			awaitingTurn = false
			awaitingMateria = true
			conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Entendido, <strong>Turno "+mensajeUsuario+"</strong>. ¿De qué materia querés saber la fecha?")))
			continue // Skip to next iteration
		}

		// PHASE 2: Awaiting Materia Name (for all options that need materia)
		if awaitingMateria {
			if currentOption == "1" {
				// Option 1: Search all dates
				handleMateriaSearch(conn, h.Repo, mensajeUsuario, "all", &awaitingDownload, &pendingCardID)
			} else if currentOption == "2" {
				// Option 2: Search by turn
				handleMateriaSearch(conn, h.Repo, mensajeUsuario, currentTurn, &awaitingDownload, &pendingCardID)
			}
			awaitingMateria = false
			currentOption = ""
			currentTurn = ""
			continue // Skip to next iteration
		}

		// PHASE 3: Menu Selection (ONLY if no other state is active)
		if currentOption == "" && !awaitingMateria && !awaitingTurn && !awaitingDownload {
			// User selecting option or typing materia directly
			if mensajeUsuario == "1" || mensajeUsuario == "a" || mensajeUsuario == "todas" {
				currentOption = "1"
				awaitingMateria = true
				conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Perfecto. ¿Qué materia estás buscando?")))
			} else if mensajeUsuario == "2" || mensajeUsuario == "b" || mensajeUsuario == "turno" {
				currentOption = "2"
				awaitingTurn = true
				conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Dale. ¿Qué número de Turno te interesa? (1 al 10)")))
			} else if mensajeUsuario == "3" || mensajeUsuario == "c" || mensajeUsuario == "disponible" || mensajeUsuario == "turnos" {
				currentOption = "3"
				// Option 3: Show future turnos (exam periods)
				sendFutureTurnos(conn, h)
				currentOption = "" // Reset immediately
			} else {
				// Direct materia search (shortcut to Option 1)
				currentOption = "1"
				handleMateriaSearch(conn, h.Repo, mensajeUsuario, "all", &awaitingDownload, &pendingCardID)
				currentOption = "" // Reset for next query
			}
			continue
		}

		// If we reach here, something unexpected happened - reset state
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Perdon, hubo un problema. Volvamos al menú principal.")))
		currentOption = ""
		awaitingMateria = false
		awaitingTurn = false
		awaitingDownload = false
		pendingCardID = ""
		sendMenuOptions(conn)
	}
}

// Helper to handle materia search with disambiguation
func handleMateriaSearch(conn *websocket.Conn, repo *repository.MesaRepository, input string, mode string, awaitingDownload *bool, pendingCardID *string) {
	// 1. Search for matching materias
	matches, err := repo.GetUniqueMaterias(input)
	if err != nil {
		log.Println("Repo Error:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Error en el servidor. Intentá de nuevo.")))
		return
	}

	// 2. Handle no matches
	if len(matches) == 0 {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("❌ No encontré ninguna materia que suene como <strong>"+input+"</strong>. Por favor, revisá el nombre e intentá de nuevo.")))
		return
	}

	// 3. Handle multiple matches (disambiguation)
	if len(matches) > 1 {
		exactMatch := false
		var selectedMateria string
		for _, m := range matches {
			if repository.Normalize(m) == repository.Normalize(input) {
				exactMatch = true
				selectedMateria = m
				break
			}
		}

		if !exactMatch {
			// Show disambiguation buttons
			sendDisambiguation(conn, matches)
			return
		}
		input = selectedMateria
	} else {
		input = matches[0] // Use exact DB name
	}

	// 4. Fetch and send results based on mode
	if mode == "all" {
		sendFullSchedule(conn, repo, input, awaitingDownload, pendingCardID)
	} else if mode == "future" {
		sendFutureDates(conn, repo, input, awaitingDownload, pendingCardID)
	} else {
		// mode is the turn name (e.g., "1° Turno")
		sendSingleTurn(conn, repo, input, mode, awaitingDownload, pendingCardID)
	}
}

// Helpers for HTML responses

func botMsg(text string) string {
	return `<div class="message-container bot"><div class="avatar">✨</div><div class="message-content"><p>` + text + `</p></div></div>`
}

func sendMenuOptions(conn *websocket.Conn) {
	html := `<div class="message-container bot"><div class="avatar">✨</div><div class="message-content">`
	html += `<p>¿Qué más necesitás saber?</p>`
	html += `<p style="margin-top: 16px; color: var(--text-secondary); line-height: 1.8;">`
	html += `<strong style="color: var(--text-primary);">1</strong> - Buscar todas las fechas de una materia<br>`
	html += `<strong style="color: var(--text-primary);">2</strong> - Buscar fecha en un turno específico<br>`
	html += `<strong style="color: var(--text-primary);">3</strong> - Ver qué turnos faltan este año`
	html += `</p></div></div>`
	conn.WriteMessage(websocket.TextMessage, []byte(html))
}

func sendDownloadPrompt(conn *websocket.Conn, cardID string) {
	conn.WriteMessage(websocket.TextMessage, []byte(botMsg("¿Querés descargar esta información como imagen? (Escribí <strong>sí</strong> o <strong>no</strong>)")))
}

func sendSingleResult(conn *websocket.Conn, resultados []models.Mesa, materia string) {
	if len(resultados) == 0 {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("No encontré nada con ese nombre en esa mesa.")))
		return
	}
	// Use the existing logic for single cards (simplified for now)
	// We can reuse the template or build string. Let's build a simple card string for speed/reliability.
	// Actually, we can reuse the previously defined logic using templates/respuesta_fragmento.html if we want,
	// but let's conform to the requested "Smart" output style.
	// For "Single Result" (Legacy), we show specific details.

	// Let's rely on `respuesta_fragmento.html` for single results as it's already styled well.
	tmpl, _ := template.ParseFiles("templates/respuesta_fragmento.html")
	var buf bytes.Buffer
	ginH := gin.H{"mensaje_usuario": materia, "resultados": resultados, "encontrado": true}
	tmpl.Execute(&buf, ginH)
	conn.WriteMessage(websocket.TextMessage, buf.Bytes())
}

func sendDisambiguation(conn *websocket.Conn, options []string) {
	html := `<div class="message-container bot"><div class="avatar">✨</div><div class="message-content"><p>Encontré varias opciones. ¿A cuál te referís?</p><div style="margin-top:12px;">`
	for _, opt := range options {
		html += `<button class="option-button" onclick="sendMessage('` + opt + `')">` + opt + `</button>`
	}
	html += `</div></div></div>`
	conn.WriteMessage(websocket.TextMessage, []byte(html))
}

func sendFullSchedule(conn *websocket.Conn, repo *repository.MesaRepository, materia string, awaitingDownload *bool, pendingCardID *string) {
	mesas, err := repo.GetFullSchedule(materia)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Error recuperando datos.")))
		return
	}

	if len(mesas) == 0 {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("❌ No encontré ninguna mesa para <strong>"+materia+"</strong>.")))
		return
	}

	// Result Card
	cardID := "card-full-" + strconv.Itoa(len(mesas))
	html := `<div class="result-card" id="` + cardID + `">`
	html += `<div style="font-size: 1.2em; font-weight: 600; color: #E3E3E3; margin-bottom: 4px;">` + materia + `</div>`
	if len(mesas) > 0 {
		html += `<div style="color: #A8C7FA; font-size: 0.9em; margin-bottom: 12px;">` + mesas[0].Carrera + `</div>`
	}

	// Table Grid
	html += `<div style="display:grid; grid-template-columns: 0.5fr 1.2fr 1fr 2fr 1.2fr; gap:8px; font-size:0.85em; color:#C4C7C5; border-top:1px solid #444; padding-top:8px;">`
	html += `<div style="font-weight:bold;">#</div><div style="font-weight:bold;">Fecha</div><div style="font-weight:bold;">Hora</div><div style="font-weight:bold;">Aula</div><div style="font-weight:bold;">Act.</div>`

	for _, m := range mesas {
		html += `<div>` + m.Turno + `</div>`
		html += `<div>` + m.Fecha + `</div>`
		html += `<div>` + m.Hora + `</div>`
		html += `<div>` + m.Aula + ` <span style="color:#888;font-size:0.8em">(` + m.Sede + `)</span></div>`
		html += `<div style="font-size:0.8em; color:#888;">` + m.FechaEdicion + `</div>`
	}
	html += `</div>` // End Grid
	html += `</div>`

	conn.WriteMessage(websocket.TextMessage, []byte(html))

	// Set download state
	*pendingCardID = cardID
	*awaitingDownload = true
	sendDownloadPrompt(conn, cardID)
}

func sendFutureDates(conn *websocket.Conn, repo *repository.MesaRepository, materia string, awaitingDownload *bool, pendingCardID *string) {
	mesas, err := repo.GetFutureDates(materia)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Error recuperando fechas.")))
		return
	}

	if len(mesas) == 0 {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("📅 Para <strong>"+materia+"</strong> ya finalizaron todos los turnos del calendario 2025.")))
		return
	}

	// Build card similar to full schedule
	cardID := "card-future-" + strconv.Itoa(len(mesas))
	html := `<div class="result-card" id="` + cardID + `">`
	html += `<div style="font-size: 1.2em; font-weight: 600; color: #E3E3E3; margin-bottom: 4px;">🗓️ Próximas Fechas: ` + materia + `</div>`
	if len(mesas) > 0 {
		html += `<div style="color: #A8C7FA; font-size: 0.9em; margin-bottom: 12px;">` + mesas[0].Carrera + `</div>`
		html += `<div style="color: #8ab4f8; font-size: 0.95em; margin-bottom: 12px;">📌 Próxima mesa: <strong>` + mesas[0].Turno + ` - ` + mesas[0].Fecha + `</strong></div>`
	}

	html += `<div style="color: #C4C7C5; font-size: 0.9em; margin-top: 16px; margin-bottom: 8px;">Resto del año:</div>`
	html += `<div style="display:grid; grid-template-columns: 0.8fr 1.2fr 1fr 2fr; gap:8px; font-size:0.85em; color:#C4C7C5; border-top:1px solid #444; padding-top:8px;">`
	html += `<div style="font-weight:bold;">#</div><div style="font-weight:bold;">Fecha</div><div style="font-weight:bold;">Hora</div><div style="font-weight:bold;">Aula</div>`

	for _, m := range mesas {
		html += `<div>` + m.Turno + `</div>`
		html += `<div>` + m.Fecha + `</div>`
		html += `<div>` + m.Hora + `</div>`
		html += `<div>` + m.Aula + ` <span style="color:#888;font-size:0.8em">(` + m.Sede + `)</span></div>`
	}
	html += `</div>`
	html += `</div>` // Close card

	conn.WriteMessage(websocket.TextMessage, []byte(html))

	// Set download state
	cardID = "card-future-" + strconv.Itoa(len(mesas))
	*pendingCardID = cardID
	*awaitingDownload = true
	sendDownloadPrompt(conn, cardID)
}

func sendSingleTurn(conn *websocket.Conn, repo *repository.MesaRepository, materia string, turnName string, awaitingDownload *bool, pendingCardID *string) {
	mesa, err := repo.GetByTurn(materia, turnName)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("❌ No encontré esa materia en el turno seleccionado.")))
		return
	}

	// Single result card
	html := `<div class="result-card" id="card-turn-` + turnName + `">`
	html += `<div style="font-size: 1.2em; font-weight: 600; color: #E3E3E3; margin-bottom: 4px;">` + mesa.Materia + `</div>`
	html += `<div style="color: #A8C7FA; font-size: 0.9em; margin-bottom: 12px;">` + mesa.Carrera + `</div>`
	html += `<div style="color: #8ab4f8; font-size: 1em; margin: 16px 0;"><strong>` + mesa.Turno + `</strong></div>`
	html += `<div style="display:grid; grid-template-columns: 1fr 1fr 2fr; gap:12px; font-size:0.9em; padding: 12px; background:#2F3031; border-radius:8px;">`
	html += `<div><span style="color:#888;">📅 Fecha:</span><br><strong>` + mesa.Fecha + `</strong></div>`
	html += `<div><span style="color:#888;">🕐 Hora:</span><br><strong>` + mesa.Hora + `</strong></div>`
	html += `<div><span style="color:#888;">🏫 Aula:</span><br><strong>` + mesa.Aula + `</strong><br><span style="font-size:0.85em; color:#888;">(` + mesa.Sede + `)</span></div>`
	html += `</div>`
	html += `<div style="margin-top: 12px; font-size: 0.8em; color: #888;">Actualizado: ` + mesa.FechaEdicion + `</div>`
	html += `</div>` // Close card

	conn.WriteMessage(websocket.TextMessage, []byte(html))

	// Set download state
	cardID := "card-turn-" + turnName
	*pendingCardID = cardID
	*awaitingDownload = true
	sendDownloadPrompt(conn, cardID)
}

func sendFutureTurnos(conn *websocket.Conn, handler *ChatHandler) {
	turnos, err := handler.ParamsRepo.GetFutureTurnos()
	if err != nil {
		log.Println("Error getting turnos:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("Error recuperando turnos.")))
		return
	}

	if len(turnos) == 0 {
		conn.WriteMessage(websocket.TextMessage, []byte(botMsg("📅 Ya finalizaron todos los turnos del calendario 2025.")))
		return
	}

	cardID := "card-turnos-" + strconv.Itoa(len(turnos))
	html := `<div class="message-container bot"><div class="avatar">✨</div><div class="message-content">`
	html += `<p><strong>📅 Turnos que faltan este año:</strong></p>`
	html += `<div class="result-card" id="` + cardID + `" style="margin-top:12px;">`

	for _, t := range turnos {
		recesoText := ""
		recesoIcon := ""
		if t.Receso {
			recesoText = " (Receso)"
			recesoIcon = "🏖️ "
		} else {
			recesoIcon = "📚 "
		}

		html += `<div style="padding:16px; border-bottom:1px solid var(--border-color); margin-bottom:8px;">`
		html += `<div style="font-weight:600; font-size:1.05em; color:var(--text-primary); margin-bottom:8px;">` + recesoIcon + t.Nombre + recesoText + `</div>`
		html += `<div style="color:var(--text-secondary); font-size:0.95em;">`
		html += `📍 <strong>Inicio:</strong> ` + formatDate(t.FechaInicio) + `<br>`
		html += `📍 <strong>Fin:</strong> ` + formatDate(t.FechaFin)
		html += `</div></div>`
	}

	html += `</div></div></div>` // Close card and containers

	conn.WriteMessage(websocket.TextMessage, []byte(html))

	// Ask if user wants to download
	sendDownloadPrompt(conn, cardID)

	// Show menu again
	sendMenuOptions(conn)
}

// Helper to format dates from YYYY-MM-DD to DD/MM/YYYY
func formatDate(dateStr string) string {
	if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
		return parsed.Format("02/01/2006")
	}
	return dateStr
}
