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

	cleanedText := h.cleanMention(activity.Text)
	log.Printf("Cleaned message: %s", cleanedText)

	context := "Tu es NEO, un assistant Microsoft 365 intégré à Teams.\n"
	if activity.From != nil && activity.From.Name != "" {
		context += "Utilisateur: " + activity.From.Name + "\n"
	}
	context += "Réponds de manière concise en français."

	// Utiliser l'ID de conversation pour le contexte
	conversationID := activity.Conversation.ID

	response, err := h.claudeService.SendMessageWithContext(cleanedText, context, conversationID, h.graphService)
	if err != nil {
		log.Printf("Claude error: %v", err)
		response = "Désolé, une erreur s'est produite."
	}

	h.sendReply(activity, response)
}

func (h *BotHandler) sendReply(activity *BotActivity, text string) {
	token, err := h.getBotToken()
	if err != nil {
		log.Printf("Error getting bot token: %v", err)
		return
	}

	replyActivity := BotActivity{
		Type:         "message",
		Text:         text,
		From:         activity.Recipient,
		Recipient:    activity.From,
		Conversation: activity.Conversation,
		ReplyToID:    activity.ID,
	}

	jsonBody, _ := json.Marshal(replyActivity)

	// URL pour répondre
	replyURL := fmt.Sprintf("%sv3/conversations/%s/activities/%s",
		activity.ServiceURL,
		activity.Conversation.ID,
		activity.ID,
	)

	req, _ := http.NewRequest("POST", replyURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending reply: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("Reply sent, status: %d", resp.StatusCode)
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

	tenantID := os.Getenv("TENANT_ID")
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	log.Printf("=== TOKEN REQUEST ===")
	log.Printf("URL: %s", tokenURL)
	log.Printf("AppID: [%s]", h.appID)
	log.Printf("AppPassword length: %d", len(h.appPassword))
	log.Printf("TenantID: [%s]", tenantID)

	resp, err := http.Post(
		tokenURL,
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		log.Printf("Token HTTP error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading token response: %v", err)
		return "", err
	}

	log.Printf("=== TOKEN RESPONSE ===")
	log.Printf("Status: %d", resp.StatusCode)
	log.Printf("Body length: %d", len(body))
	log.Printf("Body: %s", string(body))

	if len(body) == 0 {
		return "", fmt.Errorf("empty token response")
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		log.Printf("JSON unmarshal error: %v", err)
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		log.Printf("Token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
		return "", fmt.Errorf("token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	h.botToken = tokenResp.AccessToken
	h.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	log.Printf("Token obtained successfully, expires in %d seconds", tokenResp.ExpiresIn)

	return h.botToken, nil
}

func (h *BotHandler) cleanMention(text string) string {
	if idx := strings.Index(text, "</at>"); idx != -1 {
		text = strings.TrimSpace(text[idx+5:])
	}
	return text
}
