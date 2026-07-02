package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"contactless-fingerprint-backend/internal/db"
)

func main() {
	// Load .env - must run before anything else to ensure environment variables are set
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error Loading .env file")
	}

	db.Connect()

	// gin.Default() includes Logger and Recovery middleware
	router := gin.Default()

	router.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "contactless-fingerprint-backend",
		})
	})

	log.Println("Server starting on port 8080...")
	router.Run(":8080")
}
