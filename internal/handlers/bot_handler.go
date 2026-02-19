package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type BotHandler struct {
	claudeService *services.ClaudeService
	graphService  *services.GraphService
	appID         string
	appPassword   string
	botToken      string
	tokenExpiry   time.Time
	tokenMu       sync.RWMutex
}

func NewBotHandler(cs *services.ClaudeService, gs *services.GraphService) *BotHandler {
	return &BotHandler{
		claudeService: cs,
		graphService:  gs,
		appID:         os.Getenv("MICROSOFT_APP_ID"),
		appPassword:   os.Getenv("MICROSOFT_APP_PASSWORD"),
	}
}

// Activity du Bot Framework
type BotActivity struct {
	Type         string           `json:"type"`
	ID           string           `json:"id,omitempty"`
	Timestamp    string           `json:"timestamp,omitempty"`
	ServiceURL   string           `json:"serviceUrl,omitempty"`
	ChannelID    string           `json:"channelId,omitempty"`
	From         *BotAccount      `json:"from,omitempty"`
	Conversation *BotConversation `json:"conversation,omitempty"`
	Recipient    *BotAccount      `json:"recipient,omitempty"`
	Text         string           `json:"text,omitempty"`
	ReplyToID    string           `json:"replyToId,omitempty"`
	Attachments  []BotAttachment  `json:"attachments,omitempty"`
}

type BotAccount struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type BotConversation struct {
	ID               string `json:"id,omitempty"`
	ConversationType string `json:"conversationType,omitempty"`
	TenantID         string `json:"tenantId,omitempty"`
}

type BotAttachment struct {
	ContentType string      `json:"contentType,omitempty"`
	Content     interface{} `json:"content,omitempty"`
}

// POST /api/messages - Endpoint principal du Bot Framework
func (h *BotHandler) HandleMessage(c *gin.Context) {
	log.Printf("=== Bot Message Received ===")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	log.Printf("Body: %s", string(body))

	var activity BotActivity
	if err := json.Unmarshal(body, &activity); err != nil {
		log.Printf("Error parsing activity: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	log.Printf("Activity Type: %s, Text: %s", activity.Type, activity.Text)

	// Répondre 200 OK immédiatement
	c.Status(http.StatusOK)

	// Traiter en arrière-plan
	go h.processActivity(&activity)
}

func (h *BotHandler) processActivity(activity *BotActivity) {
	switch activity.Type {
	case "message":
		h.handleMessageActivity(activity)
	case "conversationUpdate":
		log.Printf("Conversation update received")
	default:
		log.Printf("Unknown activity type: %s", activity.Type)
	}
}

func (h *BotHandler) handleMessageActivity(activity *BotActivity) {
	if activity.Text == "" {
		return
	}

	// Nettoyer le message
	cleanedText := h.cleanMention(activity.Text)
	log.Printf("Cleaned message: %s", cleanedText)

	// Construire le contexte
	context := "Tu es NEO, un assistant Microsoft 365 intégré à Teams.\n"
	if activity.From != nil && activity.From.Name != "" {
		context += "Utilisateur: " + activity.From.Name + "\n"
	}
	context += "Réponds de manière concise en français."

	// Appeler Claude
	response, err := h.claudeService.SendMessageWithTools(cleanedText, context, h.graphService)
	if err != nil {
		log.Printf("Claude error: %v", err)
		response = "Désolé, une erreur s'est produite."
	}

	// Envoyer la réponse
	h.sendReply(activity, response)
}

func (h *BotHandler) sendReply(activity *BotActivity, text string) {
	token, err := h.getBotToken()
	if err != nil {
		log.Printf("Failed to get bot token: %v", err)
		return
	}

	// Construire l'URL de réponse - s'assurer que serviceUrl se termine par /
	serviceURL := strings.TrimSuffix(activity.ServiceURL, "/")
	replyURL := fmt.Sprintf("%s/v3/conversations/%s/activities/%s",
		serviceURL,
		activity.Conversation.ID,
		activity.ID,
	)

	reply := BotActivity{
		Type:         "message",
		From:         activity.Recipient,
		Recipient:    activity.From,
		Conversation: activity.Conversation,
		Text:         text,
		ReplyToID:    activity.ID,
	}

	jsonBody, _ := json.Marshal(reply)
	log.Printf("=== SENDING REPLY ===")
	log.Printf("URL: %s", replyURL)
	log.Printf("Body: %s", string(jsonBody))

	req, _ := http.NewRequest("POST", replyURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send reply: %v", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Reply sent, status: %d", resp.StatusCode)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		log.Printf("Reply error body: %s", string(respBody))
	}
}

func (h *BotHandler) getBotToken() (string, error) {
	h.tokenMu.RLock()
	if h.botToken != "" && time.Now().Before(h.tokenExpiry) {
		defer h.tokenMu.RUnlock()
		return h.botToken, nil
	}
	h.tokenMu.RUnlock()

	h.tokenMu.Lock()
	defer h.tokenMu.Unlock()

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", h.appID)
	data.Set("client_secret", h.appPassword)
	data.Set("scope", "https://api.botframework.com/.default")

	// Utiliser directement votre tenant ID
	tenantID := os.Getenv("TENANT_ID")
	if tenantID == "" {
		tenantID = "210f5bae-0956-49fb-944b-ed9648c0bd45"
	}
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	log.Printf("=== TOKEN REQUEST ===")
	log.Printf("URL: %s", tokenURL)
	log.Printf("AppID: [%s]", h.appID)
	log.Printf("TenantID: [%s]", tenantID)

	resp, err := http.Post(
		tokenURL,
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("=== TOKEN RESPONSE ===")
	log.Printf("Status: %d", resp.StatusCode)
	log.Printf("Body: %s", string(body))

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	h.botToken = tokenResp.AccessToken
	h.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	log.Printf("Token obtained successfully")

	return h.botToken, nil
}

func (h *BotHandler) cleanMention(text string) string {
	if idx := strings.Index(text, "</at>"); idx != -1 {
		text = strings.TrimSpace(text[idx+5:])
	}
	return text
}
