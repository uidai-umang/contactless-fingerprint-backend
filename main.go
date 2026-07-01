package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
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