package handlers

import (
	"bufio"
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

// Structure webhook Teams
type TeamsMessage struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	From         From   `json:"from"`
	Conversation struct {
		ID string `json:"id"`
	} `json:"conversation"`
	ServiceURL   string       `json:"serviceUrl"`
	ChannelData  *ChannelData `json:"channelData,omitempty"`
}

type From struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelData struct {
	Team    *TeamInfo    `json:"team,omitempty"`
	Channel *ChannelInfo `json:"channel,omitempty"`
}

type TeamInfo struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ChannelInfo struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type WebhookResponse struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// POST /api/webhook - Endpoint pour Outgoing Webhook Teams
func (h *TeamsBotHandler) HandleWebhook(c *gin.Context) {
	// Lire le body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		c.JSON(http.StatusBadRequest, WebhookResponse{
			Type: "message",
			Text: "Erreur de lecture",
		})
		return
	}

	log.Printf("Webhook received: %s", string(body))
	log.Printf("Headers: %v", c.Request.Header)

	// Vérifier la signature HMAC si secret configuré
	if h.webhookSecret != "" {
		if !h.verifyHMAC(c, body) {
			log.Printf("HMAC verification failed")
			c.JSON(http.StatusUnauthorized, WebhookResponse{
				Type: "message",
				Text: "Signature invalide",
			})
			return
		}
	}

	var msg TeamsMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("Error parsing message: %v", err)
		c.JSON(http.StatusOK, WebhookResponse{
			Type: "message",
			Text: "Erreur de parsing du message",
		})
		return
	}

	// Si le message est vide, retourner une réponse simple
	if msg.Text == "" {
		c.JSON(http.StatusOK, WebhookResponse{
			Type: "message",
			Text: "Bonjour! Je suis NEO, votre assistant. Comment puis-je vous aider?",
		})
		return
	}

	// Nettoyer le message (enlever la mention du bot)
	cleanedText := h.cleanMention(msg.Text)
	log.Printf("Cleaned message: %s", cleanedText)

	// Construire le contexte
	context := h.buildContext(&msg)

	// Appeler Claude
	response, err := h.claudeService.SendMessageWithTools(cleanedText, context, h.graphService)
	if err != nil {
		log.Printf("Claude error: %v", err)
		response = "Désolé, une erreur s'est produite. Réessayez plus tard."
	}

	log.Printf("Response: %s", response)

	// Répondre au webhook
	c.JSON(http.StatusOK, WebhookResponse{
		Type: "message",
		Text: response,
	})
}

func (h *TeamsBotHandler) verifyHMAC(c *gin.Context, body []byte) bool {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	// Le header peut être "HMAC <signature>" ou juste la signature
	providedMAC := strings.TrimPrefix(authHeader, "HMAC ")

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedMAC := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedMAC), []byte(providedMAC))
}

func (h *TeamsBotHandler) cleanMention(text string) string {
	// Teams ajoute <at>BotName</at> au début
	if idx := strings.Index(text, "</at>"); idx != -1 {
		text = text[idx+5:]
	}
	text = strings.ReplaceAll(text, "<at>", "")
	text = strings.ReplaceAll(text, "</at>", "")
	return strings.TrimSpace(text)
}

func (h *TeamsBotHandler) buildContext(msg *TeamsMessage) string {
	ctx := "Tu es NEO, un assistant Microsoft 365 intégré à Teams.\n"
	ctx += "Utilisateur: " + msg.From.Name + "\n"

	if msg.ChannelData != nil {
		if msg.ChannelData.Team != nil && msg.ChannelData.Team.Name != "" {
			ctx += "Équipe: " + msg.ChannelData.Team.Name + "\n"
		}
		if msg.ChannelData.Channel != nil && msg.ChannelData.Channel.Name != "" {
			ctx += "Canal: " + msg.ChannelData.Channel.Name + "\n"
		}
	}

	ctx += "Réponds de manière concise en français."
	return ctx
}
