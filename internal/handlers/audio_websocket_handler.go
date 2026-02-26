package handlers

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"sync"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket" // ✅ Remplacer golang.org/x/net/websocket
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Accepter toutes les origines
	},
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
}

type AudioSession struct {
	callID        string
	conn          *websocket.Conn // ✅ gorilla/websocket
	geminiService *services.GeminiService
	graphService  *services.GraphService
	done          chan struct{}
}

type AudioWebSocketHandler struct {
	geminiService      *services.GeminiService
	graphService       *services.GraphService
	audioBridgeService *services.AudioBridgeService
	sessions           map[string]*AudioSession
	mu                 sync.RWMutex
}

func NewAudioWebSocketHandler(
	geminiService *services.GeminiService,
	graphService *services.GraphService,
	audioBridgeService *services.AudioBridgeService,
) *AudioWebSocketHandler {
	return &AudioWebSocketHandler{
		geminiService:      geminiService,
		graphService:       graphService,
		audioBridgeService: audioBridgeService,
		sessions:           make(map[string]*AudioSession),
	}
}

func (h *AudioWebSocketHandler) HandleWebSocket(c *gin.Context) {
	callID := c.Param("callId")
	if callID == "" {
		callID = c.GetHeader("X-Call-Id")
	}
	if callID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	// ✅ Vérifier le token AVANT l'upgrade WebSocket
	wsSecret := os.Getenv("WS_SECRET")
	if wsSecret != "" {
		token := c.GetHeader("X-WS-Token")
		if token == "" {
			token = c.Query("token")
		}
		if token != wsSecret {
			log.Printf("[AudioWS] Token invalide pour callID: %s (reçu: '%s')", callID, token)
			c.Status(http.StatusForbidden)
			return
		}
	}

	log.Printf("[AudioWS] Nouvelle connexion C# pour callID: %s", callID)

	// ✅ Upgrade avec gorilla/websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[AudioWS] Erreur upgrade WebSocket: %v", err)
		return
	}
	defer conn.Close()

	session := &AudioSession{
		callID:        callID,
		conn:          conn,
		geminiService: h.geminiService,
		graphService:  h.graphService,
		done:          make(chan struct{}),
	}

	h.mu.Lock()
	h.sessions[callID] = session
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.sessions, callID)
		h.mu.Unlock()
		log.Printf("[AudioWS] Session fermée pour callID: %s", callID)
	}()

	h.handleAudioSession(session)
}

func (h *AudioWebSocketHandler) handleAudioSession(session *AudioSession) {
	log.Printf("[AudioWS] Session audio démarrée pour callID: %s", session.callID)

	audioAccumulator := make([]byte, 0, 32*1024)
	const minChunkSize = 16000

	for {
		// ✅ gorilla/websocket : ReadMessage au lieu de websocket.Message.Receive
		_, pcmChunk, err := session.conn.ReadMessage()
		if err != nil {
			log.Printf("[AudioWS] Connexion fermée pour callID %s: %v", session.callID, err)
			return
		}

		if len(pcmChunk) == 0 {
			continue
		}

		audioAccumulator = append(audioAccumulator, pcmChunk...)

		if len(audioAccumulator) < minChunkSize {
			continue
		}

		audioToProcess := make([]byte, len(audioAccumulator))
		copy(audioToProcess, audioAccumulator)
		audioAccumulator = audioAccumulator[:0]

		go func(audio []byte) {
			responseAudio, err := h.processAudioWithGemini(session, audio)
			if err != nil {
				log.Printf("[AudioWS] Erreur Gemini pour callID %s: %v", session.callID, err)
				return
			}

			if len(responseAudio) == 0 {
				return
			}

			// ✅ gorilla/websocket : WriteMessage au lieu de websocket.Message.Send
			if err := session.conn.WriteMessage(websocket.BinaryMessage, responseAudio); err != nil {
				log.Printf("[AudioWS] Erreur envoi réponse audio pour callID %s: %v", session.callID, err)
			}
		}(audioToProcess)
	}
}

func (h *AudioWebSocketHandler) processAudioWithGemini(session *AudioSession, pcmAudio []byte) ([]byte, error) {
	audioB64 := base64.StdEncoding.EncodeToString(pcmAudio)
	return h.geminiService.SendAudioMessage(audioB64, session.callID, h.graphService)
}

func (h *AudioWebSocketHandler) GetActiveSessions() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessions := make([]string, 0, len(h.sessions))
	for callID := range h.sessions {
		sessions = append(sessions, callID)
	}
	return sessions
}
