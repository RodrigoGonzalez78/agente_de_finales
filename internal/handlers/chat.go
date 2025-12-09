package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"mi-bot-unne/internal/models"
	"mi-bot-unne/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/looplab/fsm"
	"github.com/patrickmn/go-cache"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for dev
	},
}

// ChatSession almacena el estado de cada usuario
type ChatSession struct {
	FSM           *fsm.FSM
	Conn          *websocket.Conn
	Handler       *ChatHandler
	CurrentOption string
	CurrentTurn   string
	PendingCardID string
	LastInput     string
	mu            sync.Mutex
}

type ChatHandler struct {
	Repo         *repository.MesaRepository
	ParamsRepo   *repository.ParamsRepository
	SessionCache *cache.Cache
}

func NewChatHandler(repo *repository.MesaRepository, paramsRepo *repository.ParamsRepository) *ChatHandler {
	return &ChatHandler{
		Repo:         repo,
		ParamsRepo:   paramsRepo,
		SessionCache: cache.New(cache.NoExpiration, 10*time.Minute),
	}
}

func (h *ChatHandler) ShowChat(c *gin.Context) {
	c.HTML(http.StatusOK, "chat.html", nil)
}

// NewChatSession crea una nueva sesión con su máquina de estados
func NewChatSession(conn *websocket.Conn, handler *ChatHandler) *ChatSession {
	session := &ChatSession{
		Conn:    conn,
		Handler: handler,
	}

	// Definir la máquina de estados
	session.FSM = fsm.NewFSM(
		"idle", // Estado inicial
		fsm.Events{
			// Flujo principal
			{Name: "start", Src: []string{"idle"}, Dst: "menu"},
			{Name: "select_all_dates", Src: []string{"menu"}, Dst: "awaiting_materia_all"},
			{Name: "select_by_turn", Src: []string{"menu"}, Dst: "awaiting_turn"},
			{Name: "show_future_turns", Src: []string{"menu"}, Dst: "showing_turns"},
			{Name: "direct_search", Src: []string{"menu"}, Dst: "awaiting_materia_all"},

			// Submenu de turno
			{Name: "provide_turn", Src: []string{"awaiting_turn"}, Dst: "awaiting_materia_turn"},

			// Búsqueda de materia
			{Name: "provide_materia", Src: []string{"awaiting_materia_all", "awaiting_materia_turn"}, Dst: "showing_results"},
			{Name: "disambiguate", Src: []string{"awaiting_materia_all", "awaiting_materia_turn"}, Dst: "disambiguating"},
			{Name: "select_option", Src: []string{"disambiguating"}, Dst: "showing_results"},

			// Descarga
			{Name: "ask_download", Src: []string{"showing_results", "showing_turns"}, Dst: "awaiting_download"},
			{Name: "download_yes", Src: []string{"awaiting_download"}, Dst: "menu"},
			{Name: "download_no", Src: []string{"awaiting_download"}, Dst: "menu"},

			// Reset en cualquier momento
			{Name: "reset", Src: []string{"awaiting_materia_all", "awaiting_materia_turn", "awaiting_turn", "showing_results", "awaiting_download", "disambiguating", "showing_turns"}, Dst: "menu"},
			{Name: "help", Src: []string{"menu", "awaiting_materia_all", "awaiting_turn", "awaiting_materia_turn", "disambiguating"}, Dst: "menu"},
		},
		fsm.Callbacks{
			// Callbacks de entrada a estados
			"enter_menu":                  session.onEnterMenu,
			"enter_awaiting_materia_all":  session.onEnterAwaitingMateriaAll,
			"enter_awaiting_turn":         session.onEnterAwaitingTurn,
			"enter_awaiting_materia_turn": session.onEnterAwaitingMateriaTurn,
			"enter_showing_results":       session.onEnterShowingResults,
			"enter_showing_turns":         session.onEnterShowingTurns,
			"enter_awaiting_download":     session.onEnterAwaitingDownload,
			"enter_disambiguating":        session.onEnterDisambiguating,

			// Callbacks de eventos
			"after_download_yes": session.onDownloadYes,
			"after_download_no":  session.onDownloadNo,
			"after_help":         session.onHelp,
		},
	)

	return session
}

// ProcessMessage procesa el mensaje del usuario según el estado actual
func (s *ChatSession) ProcessMessage(input string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = strings.ToLower(strings.TrimSpace(input))
	s.LastInput = input

	if input == "" {
		return
	}

	currentState := s.FSM.Current()
	log.Printf("State: %s, Input: %s", currentState, input)

	ctx := context.Background()

	// Comandos globales
	if input == "menu" || input == "menú" || input == "volver" {
		s.FSM.Event(ctx, "reset")
		return
	}

	if input == "ayuda" || input == "help" {
		s.FSM.Event(ctx, "help")
		return
	}

	// Procesamiento según estado actual
	switch currentState {
	case "idle":
		s.FSM.Event(ctx, "start")

	case "menu":
		s.handleMenuInput(ctx, input)

	case "awaiting_turn":
		s.handleTurnInput(ctx, input)

	case "awaiting_materia_all", "awaiting_materia_turn":
		s.handleMateriaInput(ctx, input)

	case "disambiguating":
		// El usuario seleccionó una opción específica
		s.handleMateriaInput(ctx, input)

	case "awaiting_download":
		s.handleDownloadInput(ctx, input)
	}
}

// ============= Handlers de Input =============

func (s *ChatSession) handleMenuInput(ctx context.Context, input string) {
	switch {
	case input == "1" || input == "a" || input == "todas":
		s.FSM.Event(ctx, "select_all_dates")
	case input == "2" || input == "b" || input == "turno":
		s.FSM.Event(ctx, "select_by_turn")
	case input == "3" || input == "c" || input == "disponible" || input == "turnos":
		s.FSM.Event(ctx, "show_future_turns")
	default:
		// Búsqueda directa
		s.CurrentOption = "all"
		s.FSM.Event(ctx, "direct_search")
	}
}

func (s *ChatSession) handleTurnInput(ctx context.Context, input string) {
	turnNum, err := strconv.Atoi(input)
	if err != nil || turnNum < 1 || turnNum > 10 {
		s.sendMessage(botMsg("⚠️ Por favor ingresá un número entre 1 y 10."))
		return
	}
	s.CurrentTurn = strconv.Itoa(turnNum) + "° Turno"
	s.FSM.Event(ctx, "provide_turn")
}

func (s *ChatSession) handleMateriaInput(ctx context.Context, input string) {
	s.sendMessage(botMsg("🔍 Buscando <strong>" + input + "</strong>..."))

	// Buscar coincidencias
	matches, err := s.Handler.Repo.GetUniqueMaterias(input)
	if err != nil {
		log.Println("Error:", err)
		s.sendMessage(botMsg("❌ Error al buscar. Intentá nuevamente."))
		s.FSM.Event(ctx, "reset")
		return
	}

	if len(matches) == 0 {
		s.sendMessage(botMsg("❌ No encontré ninguna materia con ese nombre."))
		s.FSM.Event(ctx, "reset")
		return
	}

	// Múltiples coincidencias
	if len(matches) > 1 {
		exactMatch := false
		for _, m := range matches {
			if repository.Normalize(m) == repository.Normalize(input) {
				exactMatch = true
				input = m
				break
			}
		}

		if !exactMatch {
			s.showDisambiguation(matches)
			s.FSM.Event(ctx, "disambiguate")
			return
		}
	} else {
		input = matches[0]
	}

	// Realizar búsqueda según el modo
	s.performSearch(ctx, input)
}

func (s *ChatSession) handleDownloadInput(ctx context.Context, input string) {
	if input == "si" || input == "sí" || input == "s" {
		s.FSM.Event(ctx, "download_yes")
	} else {
		s.FSM.Event(ctx, "download_no")
	}
}

// ============= Callbacks de Estados =============

func (s *ChatSession) onEnterMenu(_ context.Context, e *fsm.Event) {
	s.sendMenuOptions()
}

func (s *ChatSession) onEnterAwaitingMateriaAll(ctx context.Context, e *fsm.Event) {
	s.CurrentOption = "all"
	if e.Event == "direct_search" {
		// Ya tenemos el input en LastInput, procesarlo
		s.handleMateriaInput(ctx, s.LastInput)
	} else {
		s.sendMessage(botMsg("Perfecto. ¿Qué materia estás buscando?"))
	}
}

func (s *ChatSession) onEnterAwaitingTurn(_ context.Context, e *fsm.Event) {
	s.sendMessage(botMsg("Dale. ¿Qué número de turno te interesa? (1 al 10)"))
}

func (s *ChatSession) onEnterAwaitingMateriaTurn(_ context.Context, e *fsm.Event) {
	s.sendMessage(botMsg("Perfecto, <strong>Turno " + strings.TrimSuffix(s.CurrentTurn, "° Turno") + "</strong>. ¿Qué materia buscás?"))
}

func (s *ChatSession) onEnterShowingResults(ctx context.Context, e *fsm.Event) {
	// Los resultados ya se mostraron en performSearch
	// Preguntar por descarga
	s.FSM.Event(ctx, "ask_download")
}

func (s *ChatSession) onEnterShowingTurns(ctx context.Context, e *fsm.Event) {
	s.sendFutureTurnos()
	s.FSM.Event(ctx, "ask_download")
}

func (s *ChatSession) onEnterAwaitingDownload(_ context.Context, e *fsm.Event) {
	s.sendMessage(botMsg("¿Querés guardar esta información como imagen? Escribí <strong>sí</strong> o <strong>no</strong>"))
}

func (s *ChatSession) onEnterDisambiguating(_ context.Context, e *fsm.Event) {
	// La disambiguación ya se mostró
}

func (s *ChatSession) onDownloadYes(_ context.Context, e *fsm.Event) {
	downloadHTML := `<div style="display:none;"><button id="auto-download-btn" onclick="downloadCard('` + s.PendingCardID + `'); this.remove();">Download</button></div>`
	downloadHTML += `<script>setTimeout(() => { const btn = document.getElementById('auto-download-btn'); if(btn) btn.click(); }, 100);</script>`
	downloadHTML += botMsg("¡Perfecto! Preparando descarga... 📥")
	s.sendMessage(downloadHTML)
	s.PendingCardID = ""
}

func (s *ChatSession) onDownloadNo(_ context.Context, e *fsm.Event) {
	s.sendMessage(botMsg("Entendido 👍"))
	s.PendingCardID = ""
}

func (s *ChatSession) onHelp(_ context.Context, e *fsm.Event) {
	html := `<div class="message-container bot"><div class="avatar">🤖</div><div class="message-content">`
	html += `<p><strong>💡 Comandos útiles:</strong></p>`
	html += `<p style="margin-top: 12px; color: var(--text-secondary); line-height: 1.8;">`
	html += `• Escribí el nombre de una materia para búsqueda rápida<br>`
	html += `• <strong>1</strong>, <strong>2</strong> o <strong>3</strong> para opciones del menú<br>`
	html += `• <strong>menú</strong> para ver las opciones<br>`
	html += `• <strong>ayuda</strong> para este mensaje`
	html += `</p></div></div>`
	s.sendMessage(html)
}

// ============= Helpers =============

func (s *ChatSession) performSearch(ctx context.Context, materia string) {
	var cardID string

	if s.CurrentOption == "all" {
		mesas, err := s.Handler.Repo.GetFullSchedule(materia)
		if err != nil || len(mesas) == 0 {
			s.sendMessage(botMsg("❌ No encontré información sobre esta materia."))
			s.FSM.Event(ctx, "reset")
			return
		}
		cardID = s.renderFullSchedule(mesas, materia)
	} else {
		// Búsqueda por turno
		mesa, err := s.Handler.Repo.GetByTurn(materia, s.CurrentTurn)
		if err != nil {
			s.sendMessage(botMsg("❌ No encontré esta materia en el turno seleccionado."))
			s.FSM.Event(ctx, "reset")
			return
		}
		cardID = s.renderSingleTurn(mesa)
	}

	s.PendingCardID = cardID
	s.FSM.Event(ctx, "provide_materia")
}

func (s *ChatSession) renderFullSchedule(mesas []models.Mesa, materia string) string {
	s.sendMessage(botMsg("✅ Encontré <strong>" + strconv.Itoa(len(mesas)) + " fechas</strong>:"))

	cardID := "card-full-" + strconv.Itoa(len(mesas))
	html := `<div class="result-card" id="` + cardID + `">`
	html += `<div style="font-size: 1.2em; font-weight: 600; color: #E3E3E3; margin-bottom: 4px;">` + materia + `</div>`

	if len(mesas) > 0 {
		html += `<div style="color: #A8C7FA; font-size: 0.9em; margin-bottom: 12px;">` + mesas[0].Carrera + `</div>`
	}

	html += `<div style="display:grid; grid-template-columns: 0.5fr 1.2fr 1fr 2fr 1.2fr; gap:8px; font-size:0.85em; color:#C4C7C5; border-top:1px solid #444; padding-top:8px;">`
	html += `<div style="font-weight:bold;">#</div><div style="font-weight:bold;">Fecha</div><div style="font-weight:bold;">Hora</div><div style="font-weight:bold;">Aula</div><div style="font-weight:bold;">Act.</div>`

	for _, m := range mesas {
		html += `<div>` + m.Turno + `</div>`
		html += `<div>` + m.Fecha + `</div>`
		html += `<div>` + m.Hora + `</div>`
		html += `<div>` + m.Aula + ` <span style="color:#888;font-size:0.8em">(` + m.Sede + `)</span></div>`
		html += `<div style="font-size:0.8em; color:#888;">` + m.FechaEdicion + `</div>`
	}
	html += `</div></div>`

	s.sendMessage(html)
	return cardID
}

func (s *ChatSession) renderSingleTurn(mesa models.Mesa) string {
	cardID := "card-turn-" + mesa.Turno
	html := `<div class="result-card" id="` + cardID + `">`
	html += `<div style="font-size: 1.2em; font-weight: 600; color: #E3E3E3; margin-bottom: 4px;">` + mesa.Materia + `</div>`
	html += `<div style="color: #A8C7FA; font-size: 0.9em; margin-bottom: 12px;">` + mesa.Carrera + `</div>`
	html += `<div style="color: #8ab4f8; font-size: 1em; margin: 16px 0;"><strong>` + mesa.Turno + `</strong></div>`
	html += `<div style="display:grid; grid-template-columns: 1fr 1fr 2fr; gap:12px; font-size:0.9em; padding: 12px; background:#2F3031; border-radius:8px;">`
	html += `<div><span style="color:#888;">📅 Fecha:</span><br><strong>` + mesa.Fecha + `</strong></div>`
	html += `<div><span style="color:#888;">🕐 Hora:</span><br><strong>` + mesa.Hora + `</strong></div>`
	html += `<div><span style="color:#888;">🏫 Aula:</span><br><strong>` + mesa.Aula + `</strong><br><span style="font-size:0.85em; color:#888;">(` + mesa.Sede + `)</span></div>`
	html += `</div></div>`

	s.sendMessage(html)
	return cardID
}

func (s *ChatSession) sendFutureTurnos() {
	turnos, err := s.Handler.ParamsRepo.GetFutureTurnos()
	if err != nil || len(turnos) == 0 {
		s.sendMessage(botMsg("📅 No hay turnos disponibles."))
		return
	}

	s.sendMessage(botMsg("✅ Turnos disponibles:"))

	cardID := "card-turnos-" + strconv.Itoa(len(turnos))
	html := `<div class="result-card" id="` + cardID + `">`

	for i, t := range turnos {
		icon := "📚"
		if t.Receso {
			icon = "🏖️"
		}
		borderStyle := ""
		if i < len(turnos)-1 {
			borderStyle = "border-bottom:1px solid #444;"
		}
		html += `<div style="padding:16px; ` + borderStyle + `">`
		html += `<div style="font-weight:600; color:#E3E3E3;">` + icon + ` ` + t.Nombre + `</div>`
		html += `<div style="color:#888; font-size:0.9em; margin-top:8px;">` + formatDate(t.FechaInicio) + ` - ` + formatDate(t.FechaFin) + `</div>`
		html += `</div>`
	}
	html += `</div>`

	s.sendMessage(html)
	s.PendingCardID = cardID
}

func (s *ChatSession) showDisambiguation(options []string) {
	html := `<div class="message-container bot"><div class="avatar">🤖</div><div class="message-content">`
	html += `<p>Encontré varias opciones. ¿Cuál buscás?</p><div style="margin-top:12px;">`
	for _, opt := range options {
		html += `<button class="option-button" onclick="sendMessage('` + opt + `')">` + opt + `</button>`
	}
	html += `</div></div></div>`
	s.sendMessage(html)
}

func (s *ChatSession) sendMenuOptions() {
	// 1. Abrimos el contenedor principal del mensaje del bot y el avatar
	html := `<div class="message-container bot"><div class="avatar">🤖</div>`

	// 2. Insertamos el HTML exacto que diseñaste
	html += `<div class="message-content">`
	html += `<p><strong>¿Qué necesitás saber? </strong> </p>`
	html += `<p style="margin-top: 16px; color: var(--text-secondary); line-height: 1.8;">`
	html += `<strong style="color: var(--text-primary);">1</strong> - Buscar todas las fechas de una materia<br>`
	html += `<strong style="color: var(--text-primary);">2</strong> - Buscar fecha en un turno específico<br>`
	html += `<strong style="color: var(--text-primary);">3</strong> - Ver qué turnos faltan este año`
	html += `</p>`
	html += `<p style="margin-top: 12px; color: var(--text-tertiary); font-size: 13px;">`
	html += `Escribí el número o el nombre de una materia para comenzar.`
	html += `</p>`
	html += `</div>` // Cerramos message-content

	// 3. Cerramos el contenedor principal
	html += `</div>`

	s.sendMessage(html)
}
func (s *ChatSession) sendMessage(html string) {
	s.Conn.WriteMessage(websocket.TextMessage, []byte(html))
}

// ============= WebSocket Handler =============

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("Failed to upgrade:", err)
		return
	}
	defer conn.Close()

	// Crear sesión para este usuario
	session := NewChatSession(conn, h)

	// Iniciar la conversación
	ctx := context.Background()
	session.FSM.Event(ctx, "start")

	// Loop principal
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read Error:", err)
			break
		}
		session.ProcessMessage(string(msg))
	}
}

// ============= Helpers globales =============

func botMsg(text string) string {
	return `<div class="message-container bot"><div class="avatar">🤖</div><div class="message-content"><p>` + text + `</p></div></div>`
}

func formatDate(dateStr string) string {
	if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
		return parsed.Format("02/01/2006")
	}
	return dateStr
}
