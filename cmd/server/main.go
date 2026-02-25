package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"microsoft_connector/config"
	"microsoft_connector/internal/handlers"
	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

// Self-ping pour éviter le cold start de Render
func startSelfPing(url string) {
	go func() {
		// Attendre que le serveur démarre
		time.Sleep(10 * time.Second)

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		client := &http.Client{Timeout: 10 * time.Second}

		for range ticker.C {
			resp, err := client.Get(url + "/health")
			if err != nil {
				log.Printf("Self-ping failed: %v", err)
			} else {
				resp.Body.Close()
				log.Printf("Self-ping OK")
			}
		}
	}()
}

func main() {
	cfg := config.Load()

	port := cfg.Port
	if port == "" {
		port = "10000"
	}
	log.Printf("🚀 Starting NEO Bot (Gemini) on port %s", port)

	// Self-ping pour Render
	renderURL := os.Getenv("RENDER_EXTERNAL_URL")
	if renderURL != "" {
		log.Printf("☁️ Render detected, starting self-ping to %s", renderURL)
		startSelfPing(renderURL)
	}

	// ===== Services =====
	authService := services.NewAuthService(cfg)
	graphService := services.NewGraphService(authService)
	geminiService := services.NewGeminiService(cfg.GeminiAPIKey)

	// ===== Handler =====
	botHandler := handlers.NewBotHandler(geminiService, graphService)

	// ===== Routes =====
	r := gin.Default()

	// Route racine
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "NEO - Microsoft Teams Bot",
			"model":   "gemini-2.0-flash",
			"version": "2.0",
		})
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Bot Framework endpoint
	r.POST("/api/messages", botHandler.HandleMessage)

	addr := "0.0.0.0:" + port
	log.Printf("NEO Bot ready on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("❌ Server failed:", err)
	}
}
