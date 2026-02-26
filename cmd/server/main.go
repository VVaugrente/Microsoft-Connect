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
	log.Printf("Starting NEO Bot on port %s", port)

	renderURL := os.Getenv("RENDER_EXTERNAL_URL")
	if renderURL != "" {
		log.Printf("Render detected, starting self-ping to %s", renderURL)
		startSelfPing(renderURL)
	}

	// ===== Services =====
	authService := services.NewAuthService(cfg)
	graphService := services.NewGraphService(authService)
	geminiService := services.NewGeminiService(cfg.GeminiAPIKey)
	audioBridgeService := services.NewAudioBridgeService(cfg.AudioBridgeURL)

	// ===== Handlers =====
	botHandler := handlers.NewBotHandler(geminiService, graphService, audioBridgeService)
	audioWSHandler := handlers.NewAudioWebSocketHandler(geminiService, graphService, audioBridgeService)

	// Check C# bridge
	if audioBridgeService.IsHealthy() {
		log.Printf("✅ C# Audio Bridge connecté: %s", cfg.AudioBridgeURL)
	} else {
		log.Printf("⚠️  C# Audio Bridge non disponible (normal si pas encore lancé): %s", cfg.AudioBridgeURL)
	}

	// ===== Routes =====
	r := gin.Default()

	// Route racine
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "NEO", "version": "2.0"})
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "healthy",
			"audio_bridge": audioBridgeService.IsHealthy(),
		})
	})

	// Messages texte Teams (existant)
	r.POST("/api/messages", botHandler.HandleMessage)

	// WebSocket audio - le C# se connecte ici avec le callId
	r.GET("/ws/audio/:callId", audioWSHandler.HandleWebSocket)

	// Debug appels actifs
	r.GET("/api/calls", func(c *gin.Context) {
		calls, err := audioBridgeService.GetActiveCalls()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"calls":          calls,
			"activeSessions": audioWSHandler.GetActiveSessions(),
		})
	})

	addr := "0.0.0.0:" + port
	log.Printf("NEO Bot ready → %s", addr)
	log.Printf("WebSocket audio → wss://<host>/ws/audio/:callId")

	if err := r.Run(addr); err != nil {
		log.Fatal("❌ Server failed:", err)
	}
}
