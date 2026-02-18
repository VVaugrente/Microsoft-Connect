package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

// Activity - Format Bot Framework utilisé par Teams
type Activity struct {
	Type         string               `json:"type"`
	ID           string               `json:"id,omitempty"`
	Timestamp    string               `json:"timestamp,omitempty"`
	ServiceURL   string               `json:"serviceUrl,omitempty"`
	ChannelID    string               `json:"channelId,omitempty"`
	From         *ChannelAccount      `json:"from,omitempty"`
	Conversation *ConversationAccount `json:"conversation,omitempty"`
	Recipient    *ChannelAccount      `json:"recipient,omitempty"`
	Text         string               `json:"text,omitempty"`
	ReplyToID    string               `json:"replyToId,omitempty"`
	Attachments  []Attachment         `json:"attachments,omitempty"`
	ChannelData  interface{}          `json:"channelData,omitempty"`
}

type ChannelAccount struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type ConversationAccount struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Attachment struct {
	ContentType string      `json:"contentType,omitempty"`
	Content     interface{} `json:"content,omitempty"`
}

// POST /api/webhook - Endpoint pour Outgoing Webhook Teams
func (h *TeamsBotHandler) HandleWebhook(c *gin.Context) {
	log.Printf("=== Webhook POST received ===")
	log.Printf("Content-Type: %s", c.GetHeader("Content-Type"))
	log.Printf("Authorization: %s", c.GetHeader("Authorization"))

	// Lire le body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		c.JSON(http.StatusOK, Activity{Type: "message", Text: "OK"})
		return
	}

	log.Printf("Body: %s", string(body))

	// Body vide = validation
	if len(body) == 0 {
		log.Printf("Empty body - validation request")
		c.JSON(http.StatusOK, Activity{Type: "message", Text: "OK"})
		return
	}

	// Parser l'Activity
	var activity Activity
	if err := json.Unmarshal(body, &activity); err != nil {
		log.Printf("Error parsing activity: %v", err)
		c.JSON(http.StatusOK, Activity{Type: "message", Text: "OK"})
		return
	}

	log.Printf("Activity Type: %s, Text: %s", activity.Type, activity.Text)

	// Vérifier HMAC si secret configuré
	if h.webhookSecret != "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && !h.verifyHMAC(authHeader, body) {
			log.Printf("HMAC verification failed")
			// On continue quand même pour le debug
		}
	}

	// Répondre immédiatement pour éviter le timeout
	responseText := "Bonjour! Je suis NEO."

	if activity.Text != "" {
		// Nettoyer le message (enlever la mention du bot)
		cleanedText := h.cleanMention(activity.Text)
		log.Printf("Cleaned message: %s", cleanedText)

		if cleanedText != "" {
			// Construire le contexte
			context := h.buildContext(&activity)

			// Appeler Claude
			response, err := h.claudeService.SendMessageWithTools(cleanedText, context, h.graphService)
			if err != nil {
				log.Printf("Claude error: %v", err)
				responseText = "Désolé, une erreur s'est produite."
			} else {
				responseText = response
			}
		}
	}

	log.Printf("Response: %s", responseText)

	// Réponse au format Bot Framework Activity
	c.JSON(http.StatusOK, Activity{
		Type: "message",
		Text: responseText,
	})
}

func (h *TeamsBotHandler) verifyHMAC(authHeader string, body []byte) bool {
	// Le header est "HMAC <base64signature>"
	providedMAC := strings.TrimPrefix(authHeader, "HMAC ")

	// Le secret est en base64
	secretBytes, err := base64.StdEncoding.DecodeString(h.webhookSecret)
	if err != nil {
		log.Printf("Error decoding webhook secret: %v", err)
		secretBytes = []byte(h.webhookSecret)
	}

	mac := hmac.New(sha256.New, secretBytes)
	mac.Write(body)
	expectedMAC := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	log.Printf("Provided HMAC: %s", providedMAC)
	log.Printf("Expected HMAC: %s", expectedMAC)

	return hmac.Equal([]byte(expectedMAC), []byte(providedMAC))
}

func (h *TeamsBotHandler) cleanMention(text string) string {
	// Teams ajoute <at>BotName</at> au début du message
	// Format: "<at>NEO</at> message"
	if idx := strings.Index(text, "</at>"); idx != -1 {
		text = strings.TrimSpace(text[idx+5:])
	}
	return text
}

func (h *TeamsBotHandler) buildContext(activity *Activity) string {
	ctx := "Tu es NEO, un assistant Microsoft 365 intégré à Teams.\n"

	if activity.From != nil && activity.From.Name != "" {
		ctx += "Utilisateur: " + activity.From.Name + "\n"
	}

	ctx += "Réponds de manière concise en français."
	return ctx
}
