package main

import (
	"log"

	"mi-bot-unne/internal/database"
	"mi-bot-unne/internal/handlers"
	"mi-bot-unne/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	// Inicializar Base de Datos
	db, err := database.InitDB("./mesas.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Inicializar Repositorio
	mesaRepo := repository.NewMesaRepository(db)
	paramsRepo := repository.NewParamsRepository(db)

	// Inicializar Handlers
	chatHandler := handlers.NewChatHandler(mesaRepo, paramsRepo)
	authHandler := handlers.NewAuthHandler()
	adminHandler := handlers.NewAdminHandler(mesaRepo, paramsRepo)

	// Configurar Gin
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// Rutas Públicas
	r.GET("/", chatHandler.ShowChat)
	r.POST("/chat", chatHandler.HandleMessage) // Keep HTTP fallback if needed, or remove
	r.GET("/ws", chatHandler.HandleWebSocket)

	// Rutas de Autenticación
	r.GET("/login", authHandler.ShowLogin)
	r.POST("/do-login", authHandler.Login)
	r.GET("/logout", authHandler.Logout)

	// Rutas Protegidas (Admin)
	adminGroup := r.Group("/admin")
	adminGroup.Use(handlers.AuthMiddleware())
	{
		adminGroup.GET("", adminHandler.ShowDashboard)
		adminGroup.GET("/config", adminHandler.ShowParams) // New config page
		adminGroup.POST("/guardar", adminHandler.CreateMesa)
		adminGroup.GET("/borrar/:id", adminHandler.DeleteMesa)
		adminGroup.POST("/materias", adminHandler.StoreMateria)
		adminGroup.POST("/carreras", adminHandler.StoreCarrera) // New carrera handler
		adminGroup.POST("/sedes", adminHandler.StoreSede)
		adminGroup.POST("/aulas", adminHandler.StoreAula)

		adminGroup.GET("/api/aulas", adminHandler.GetAulas)

		adminGroup.POST("/turnos", adminHandler.StoreTurnoConfig)
		adminGroup.POST("/turnos/update/:id", adminHandler.UpdateTurnoConfig)
		adminGroup.GET("/turnos/delete/:id", adminHandler.DeleteTurnoConfig)

		// Generic Config CRUD
		adminGroup.GET("/config/edit/:type/:id", adminHandler.ShowEditParam)
		adminGroup.POST("/config/update/:type/:id", adminHandler.UpdateParam)
		adminGroup.GET("/config/delete/:type/:id", adminHandler.DeleteParam)
	}

	// Iniciar servidor
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
