package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type TeamsBotHandler struct {
	claudeService *services.ClaudeService
	graphService  *services.GraphService
	webhookSecret string
}

func NewTeamsBotHandler(cs *services.ClaudeService, gs *services.GraphService, secret string) *TeamsBotHandler {
	return &TeamsBotHandler{
		claudeService: cs,
		graphService:  gs,
		webhookSecret: secret,
	}
}

type TeamsMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
	From struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"from"`
	Conversation struct {
		ID string `json:"id"`
	} `json:"conversation"`
	ChannelData *struct {
		Team *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"team,omitempty"`
		Channel *struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"channel,omitempty"`
	} `json:"channelData,omitempty"`
}

type WebhookResponse struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// POST /api/webhook
func (h *TeamsBotHandler) HandleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		c.JSON(http.StatusOK, WebhookResponse{Type: "message", Text: "OK"})
		return
	}

	log.Printf("=== WEBHOOK RECEIVED ===")
	log.Printf("Headers: %v", c.Request.Header)
	log.Printf("Body: %s", string(body))
	log.Printf("========================")

	// Body vide = test de validation
	if len(body) == 0 {
		log.Printf("Empty body - validation test")
		c.JSON(http.StatusOK, WebhookResponse{Type: "message", Text: "OK"})
		return
	}

	var msg TeamsMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Parse error: %v", err)
		c.JSON(http.StatusOK, WebhookResponse{Type: "message", Text: "OK"})
		return
	}

	// Pas de texte = test de validation
	if msg.Text == "" {
		log.Printf("No text - validation test")
		c.JSON(http.StatusOK, WebhookResponse{Type: "message", Text: "OK"})
		return
	}

	// Message réel
	cleanedText := h.cleanMention(msg.Text)
	log.Printf("Processing message: %s", cleanedText)

	context := h.buildContext(&msg)

	response, err := h.claudeService.SendMessageWithTools(cleanedText, context, h.graphService)
	if err != nil {
		log.Printf("Claude error: %v", err)
		response = "Erreur: " + err.Error()
	}

	log.Printf("Response: %s", response)
	c.JSON(http.StatusOK, WebhookResponse{Type: "message", Text: response})
}

func (h *TeamsBotHandler) cleanMention(text string) string {
	if idx := strings.Index(text, "</at>"); idx != -1 {
		text = text[idx+5:]
	}
	text = strings.ReplaceAll(text, "<at>", "")
	text = strings.ReplaceAll(text, "</at>", "")
	return strings.TrimSpace(text)
}

func (h *TeamsBotHandler) buildContext(msg *TeamsMessage) string {
	ctx := "Tu es un assistant Microsoft 365 intégré à Teams.\n"
	ctx += "Utilisateur: " + msg.From.Name + "\n"
	ctx += "Réponds de manière concise en français."
	return ctx
}
