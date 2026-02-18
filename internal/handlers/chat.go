package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	claudeService    *services.ClaudeService
	graphService     *services.GraphService
	teamsChatService *services.TeamsChatService
}

func NewChatHandler(cs *services.ClaudeService, gs *services.GraphService) *ChatHandler {
	return &ChatHandler{
		claudeService:    cs,
		graphService:     gs,
		teamsChatService: services.NewTeamsChatService(gs),
	}
}

type ChatRequest struct {
	Message string `json:"message"`
	UserID  string `json:"user_id,omitempty"` // Pour envoyer à un utilisateur spécifique
}

type DirectMessageRequest struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ChannelMessageRequest struct {
	TeamID    string `json:"team_id"`
	ChannelID string `json:"channel_id"`
	Message   string `json:"message"`
}

// POST /api/chat - Chat avec l'IA
func (h *ChatHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	context := `Tu es un assistant Microsoft 365.
Tu peux aider avec les emails, calendrier, Teams, utilisateurs.
Réponds de manière concise en français.`

	response, err := h.claudeService.SendMessageWithTools(req.Message, context, h.graphService)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"response": response})
}

// POST /api/chat/direct - Envoyer un message direct à un utilisateur
func (h *ChatHandler) SendDirectMessage(c *gin.Context) {
	var req DirectMessageRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.teamsChatService.SendDirectMessage(req.UserID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}

// POST /api/chat/channel - Envoyer un message dans un canal
func (h *ChatHandler) SendChannelMessage(c *gin.Context) {
	var req ChannelMessageRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.teamsChatService.SendChannelMessage(req.TeamID, req.ChannelID, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sent"})
}
