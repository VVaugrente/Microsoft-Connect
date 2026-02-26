package handlers

import (
	"encoding/base64"
	"log"
	"net/http"
	"sync"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

// AudioSession représente un appel vocal actif
type AudioSession struct {
	callID        string
	conn          *websocket.Conn
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

// HandleWebSocket - Le C# se connecte ici pour streamer l'audio
// GET /ws/audio/:callId
func (h *AudioWebSocketHandler) HandleWebSocket(c *gin.Context) {
	callID := c.Param("callId")
	if callID == "" {
		// Fallback: lire depuis le header X-Call-Id envoyé par GoAudioBridge.cs
		callID = c.GetHeader("X-Call-Id")
	}
	if callID == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	log.Printf("[AudioWS] Nouvelle connexion C# pour callID: %s", callID)

	websocket.Handler(func(ws *websocket.Conn) {
		session := &AudioSession{
			callID:        callID,
			conn:          ws,
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
	}).ServeHTTP(c.Writer, c.Request)
}

// handleAudioSession - Boucle principale audio : C# → GO → Gemini → GO → C#
func (h *AudioWebSocketHandler) handleAudioSession(session *AudioSession) {
	log.Printf("[AudioWS] Session audio démarrée pour callID: %s", session.callID)

	// Accumulateur audio (Gemini a besoin de chunks suffisamment grands)
	audioAccumulator := make([]byte, 0, 32*1024)
	const minChunkSize = 16000 // 500ms de PCM 16kHz 16bit mono

	for {
		// Recevoir chunk audio PCM du C#
		var pcmChunk []byte
		if err := websocket.Message.Receive(session.conn, &pcmChunk); err != nil {
			log.Printf("[AudioWS] Connexion fermée pour callID %s: %v", session.callID, err)
			return
		}

		if len(pcmChunk) == 0 {
			continue
		}

		// Accumuler jusqu'à avoir assez de données pour Gemini
		audioAccumulator = append(audioAccumulator, pcmChunk...)

		if len(audioAccumulator) < minChunkSize {
			continue
		}

		// Copier et reset l'accumulateur
		audioToProcess := make([]byte, len(audioAccumulator))
		copy(audioToProcess, audioAccumulator)
		audioAccumulator = audioAccumulator[:0]

		// Traitement async : envoi à Gemini + réponse vers C#
		go func(audio []byte) {
			responseAudio, err := h.processAudioWithGemini(session, audio)
			if err != nil {
				log.Printf("[AudioWS] Erreur Gemini pour callID %s: %v", session.callID, err)
				return
			}

			if len(responseAudio) == 0 {
				return
			}

			// Renvoyer l'audio Gemini vers le C# via WebSocket
			if err := websocket.Message.Send(session.conn, responseAudio); err != nil {
				log.Printf("[AudioWS] Erreur envoi réponse audio pour callID %s: %v", session.callID, err)
			}
		}(audioToProcess)
	}
}

// processAudioWithGemini - Envoie l'audio à Gemini et récupère la réponse audio
func (h *AudioWebSocketHandler) processAudioWithGemini(session *AudioSession, pcmAudio []byte) ([]byte, error) {
	//Encoder en base64 pour Gemini Live API
	audioB64 := base64.StdEncoding.EncodeToString(pcmAudio)

	// Appeler Gemini Live avec l'audio
	responseAudio, err := h.geminiService.SendAudioMessage(audioB64, session.callID, h.graphService)
	if err != nil {
		return nil, err
	}

	return responseAudio, nil
}

// GetActiveSessions - Retourne les sessions actives (pour debug)
func (h *AudioWebSocketHandler) GetActiveSessions() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	sessions := make([]string, 0, len(h.sessions))
	for callID := range h.sessions {
		sessions = append(sessions, callID)
	}
	return sessions
}
