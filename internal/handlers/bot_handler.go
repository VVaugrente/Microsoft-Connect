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
	geminiService      *services.GeminiService
	graphService       *services.GraphService
	audioBridgeService *services.AudioBridgeService
	appID              string
	appPassword        string
	botToken           string
	tokenExpiry        time.Time
	tokenMu            sync.RWMutex
}

func NewBotHandler(gs *services.GeminiService, graphService *services.GraphService, audioBridgeService *services.AudioBridgeService) *BotHandler {
	return &BotHandler{
		geminiService:      gs,
		graphService:       graphService,
		audioBridgeService: audioBridgeService,
		appID:              os.Getenv("MICROSOFT_APP_ID"),
		appPassword:        os.Getenv("MICROSOFT_APP_PASSWORD"),
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
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	AadObjectId string `json:"aadObjectId,omitempty"`
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

	var activity BotActivity
	if err := json.Unmarshal(body, &activity); err != nil {
		log.Printf("Error parsing activity: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	// Répondre 200 OK immédiatement
	c.Status(http.StatusOK)

	// Traiter en arrière-plan
	go h.processActivity(&activity)
}

func (h *BotHandler) processActivity(activity *BotActivity) {
	switch activity.Type {
	case "message":
		h.handleMessageActivity(activity)
	default:
		log.Printf("Unknown activity type: %s", activity.Type)
	}
}

func (h *BotHandler) handleMessageActivity(activity *BotActivity) {
	if activity.Text == "" {
		return
	}

	cleanedText := h.cleanMention(activity.Text)

	if isCreateAndJoinCommand(cleanedText) {
		h.handleCreateAndJoinRequest(activity)
		return
	}
	if isJoinVoiceCommand(cleanedText) {
		h.handleVoiceJoinRequest(activity, cleanedText)
		return
	}
	if isLeaveVoiceCommand(cleanedText) {
		h.handleVoiceLeaveRequest(activity)
		return
	}

	// ← MANQUAIT : traitement texte normal via Gemini
	conversationID := activity.Conversation.ID
	userID := ""
	if activity.From != nil {
		userID = activity.From.AadObjectId
	}

	context := h.buildSystemContext(userID)

	response, err := h.geminiService.SendMessageWithContext(cleanedText, context, conversationID, h.graphService)
	if err != nil {
		log.Printf("Error calling Gemini: %v", err)
		h.sendReply(activity, "❌ Erreur lors du traitement de votre message.")
		return
	}

	h.sendReply(activity, response)
}

func isCreateAndJoinCommand(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return lower == "appel" ||
		lower == "vocal" ||
		lower == "démarre un appel" ||
		lower == "start call" ||
		lower == "rejoins"
}

func (h *BotHandler) handleCreateAndJoinRequest(activity *BotActivity) {
	h.sendReply(activity, "📅 Création d'une réunion, un instant...")

	userID := ""
	if activity.From != nil {
		userID = activity.From.AadObjectId
	}
	if userID == "" {
		h.sendReply(activity, "❌ Impossible d'identifier l'utilisateur.")
		return
	}

	now := time.Now().UTC()

	// ✅ Utiliser /events au lieu de /onlineMeetings → pas besoin de policy
	meetingBody := map[string]any{
		"subject": "Appel avec NEO",
		"start": map[string]string{
			"dateTime": now.Format(time.RFC3339),
			"timeZone": "UTC",
		},
		"end": map[string]string{
			"dateTime": now.Add(1 * time.Hour).Format(time.RFC3339),
			"timeZone": "UTC",
		},
		"isOnlineMeeting":       true,
		"onlineMeetingProvider": "teamsForBusiness",
	}

	result, err := h.graphService.Post("/users/"+userID+"/events", meetingBody)
	if err != nil {
		h.sendReply(activity, fmt.Sprintf("❌ Impossible de créer la réunion: %v", err))
		return
	}

	// Extraire le lien Teams et l'ID de la réunion en ligne
	onlineMeeting, _ := result["onlineMeeting"].(map[string]any)
	joinURL := ""
	onlineMeetingID := ""
	if onlineMeeting != nil {
		joinURL, _ = onlineMeeting["joinUrl"].(string)
		onlineMeetingID, _ = onlineMeeting["id"].(string)
	}

	if joinURL == "" {
		h.sendReply(activity, "❌ Lien de réunion introuvable.")
		return
	}

	// ✅ Configurer le lobby pour que tout le monde bypass
	if onlineMeetingID != "" {
		lobbyBody := map[string]any{
			"lobbyBypassSettings": map[string]any{
				"scope":                 "everyone", // ← tout le monde bypass le lobby
				"isDialInBypassEnabled": true,
			},
		}
		_, err = h.graphService.Patch("/users/"+userID+"/onlineMeetings/"+onlineMeetingID, lobbyBody)
		if err != nil {
			log.Printf("[AudioBridge] Impossible de configurer le lobby: %v", err)
			// ← pas bloquant, on continue quand même
		} else {
			log.Printf("[AudioBridge] ✅ Lobby configuré: everyone bypass")
		}
	}

	h.sendReply(activity, fmt.Sprintf(
		"✅ Réunion créée ! Rejoins d'abord, NEO arrive dans 10 secondes.\n\n[🎙️ Rejoindre l'appel avec NEO](%s)", joinURL,
	))

	time.Sleep(10 * time.Second)

	_, err = h.audioBridgeService.JoinCall(joinURL, "NEO")
	if err != nil {
		log.Printf("[AudioBrigde] Erreur JoinCall: %v", err)
		h.sendReply(activity, fmt.Sprintf("❌ NEO n'a pas pu rejoindre: %v", err))
		return
	}

	h.sendReply(activity, "🎙️ NEO a rejoint la réunion !")
}

func isJoinVoiceCommand(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "rejoins la réunion") ||
		strings.Contains(lower, "rejoins l'appel") ||
		strings.Contains(lower, "join meeting") ||
		strings.Contains(lower, "teams.microsoft.com/l/meetup-join/")
}

func isLeaveVoiceCommand(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "quitte la réunion") ||
		strings.Contains(lower, "quitte l'appel") ||
		strings.Contains(lower, "leave meeting")
}

func (h *BotHandler) handleVoiceJoinRequest(activity *BotActivity, text string) {
	joinURL := extractMeetingURL(text)
	if joinURL == "" {
		h.sendReply(activity, "❌ Partagez le lien Teams de la réunion.")
		return
	}

	h.sendReply(activity, "🎙️ Je rejoins la réunion, un instant...")

	resp, err := h.audioBridgeService.JoinCall(joinURL, "NEO")
	if err != nil {
		log.Printf("[BotHandler] Erreur JoinCall: %v", err)
		h.sendReply(activity, fmt.Sprintf("❌ Impossible de rejoindre: %v", err))
		return
	}

	h.sendReply(activity, fmt.Sprintf("✅ J'ai rejoint la réunion ! Je vous écoute. (ID: %s)", resp.CallID))
}

func (h *BotHandler) handleVoiceLeaveRequest(activity *BotActivity) {
	calls, err := h.audioBridgeService.GetActiveCalls()
	if err != nil || len(calls) == 0 {
		h.sendReply(activity, "ℹ️ Je ne suis dans aucune réunion.")
		return
	}

	if err := h.audioBridgeService.LeaveCall(calls[0].ThreadID); err != nil {
		h.sendReply(activity, fmt.Sprintf("❌ Erreur: %v", err))
		return
	}
	h.sendReply(activity, "👋 J'ai quitté la réunion.")
}

func extractMeetingURL(text string) string {
	for _, word := range strings.Fields(text) {
		if strings.Contains(word, "teams.microsoft.com/l/meetup-join/") {
			return word
		}
	}
	return ""
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

	resp, err := http.Post(
		tokenURL,
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("=== TOKEN RESPONSE Status: %d ===", resp.StatusCode)

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
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("token error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	h.botToken = tokenResp.AccessToken
	h.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	log.Printf("Bot token obtained, expires in %d seconds", tokenResp.ExpiresIn)
	return h.botToken, nil
}

func (h *BotHandler) cleanMention(text string) string {
	if idx := strings.Index(text, "</at>"); idx != -1 {
		text = strings.TrimSpace(text[idx+5:])
	}
	return text
}

func (h *BotHandler) buildSystemContext(userID string) string {
	return fmt.Sprintf(`Tu es NEO, un assistant IA Microsoft 365 intégré dans Teams.
Tu aides les utilisateurs avec leurs emails, calendrier, réunions et tâches.
Réponds toujours en français de manière concise et professionnelle.
Utilise les outils disponibles pour accéder aux données Microsoft 365.
ID utilisateur courant : %s`, userID)
}
