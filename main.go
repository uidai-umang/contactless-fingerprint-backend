package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"contactless-fingerprint-backend/internal/db"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error Loading .env file")
	}

	db.Connect()

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
