package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"contactless-fingerprint-backend/internal/db"
	"contactless-fingerprint-backend/internal/handler"
	"contactless-fingerprint-backend/internal/repository"
	"contactless-fingerprint-backend/internal/service"
)

func main() {
	// Load .env - must run before anything else to ensure environment variables are set
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error Loading .env file")
	}

	// Connect to PostgreSQL
	db.Connect()

	// gin.Default() includes Logger and Recovery middleware
	router := gin.Default()

	// ── Dependency injection ──────────────────────────────────────────────
	// Repositories — talk to DB
	residentRepo := repository.NewResidentRepository(db.DB)
	sessionRepo := repository.NewSessionRepository(db.DB)
	captureRepo := repository.NewCaptureRepository(db.DB)

	// Services — business logic
	residentService := service.NewResidentService(residentRepo, captureRepo)
	sessionService := service.NewSessionService(sessionRepo)
	captureService := service.NewCaptureService(captureRepo, sessionRepo)

	// Handlers — HTTP layer
	residentHandler := handler.NewResidentHandler(residentService)
	sessionHandler := handler.NewSessionHandler(sessionService)
	captureHandler := handler.NewCaptureHandler(captureService)

	// ── Routes ───────────────────────────────────────────────────────────
	api := router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"status":  "ok",
				"service": "contactless-fingerprint-backend",
			})
		})

		// Resident routes
		api.POST("/residents/lookup", residentHandler.LookupOrCreate)

		// Session routes
		api.POST("/sessions", sessionHandler.Create)
		api.POST("/sessions/close", sessionHandler.Close)

		// Capture routes
		api.POST("/captures", captureHandler.Upload)
		api.POST("/captures/batch", captureHandler.BatchUpload)

		// Dev/test only — reset resident data
		api.DELETE("/dev/reset", residentHandler.Reset)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on port 8080...")
	router.Run(":8080")
}
