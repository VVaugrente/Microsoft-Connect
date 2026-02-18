package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type TeamsBetaHandler struct {
	graphService *services.GraphService
}

func NewTeamsBetaHandler(gs *services.GraphService) *TeamsBetaHandler {
	return &TeamsBetaHandler{graphService: gs}
}

// GET /api/beta/teams/:groupId/channels/:channelId/messages
func (h *TeamsBetaHandler) GetChannelMessages(c *gin.Context) {
	groupId := c.Param("groupId")
	channelId := c.Param("channelId")
	result, err := h.graphService.GetBeta("/teams/" + groupId + "/channels/" + channelId + "/messages")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/beta/teams/:groupId/channels/:channelId/messages/:messageId/replies
func (h *TeamsBetaHandler) GetMessageReplies(c *gin.Context) {
	groupId := c.Param("groupId")
	channelId := c.Param("channelId")
	messageId := c.Param("messageId")
	result, err := h.graphService.GetBeta("/teams/" + groupId + "/channels/" + channelId + "/messages/" + messageId + "/replies")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/beta/teams/:groupId/channels/:channelId/messages/:messageId/replies/:replyId
func (h *TeamsBetaHandler) GetMessageReply(c *gin.Context) {
	groupId := c.Param("groupId")
	channelId := c.Param("channelId")
	messageId := c.Param("messageId")
	replyId := c.Param("replyId")
	result, err := h.graphService.GetBeta("/teams/" + groupId + "/channels/" + channelId + "/messages/" + messageId + "/replies/" + replyId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/beta/me/apps
func (h *TeamsBetaHandler) GetMyInstalledApps(c *gin.Context) {
	result, err := h.graphService.GetBeta("/me/teamwork/installedApps?$expand=teamsApp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/beta/chats/:chatId/members
func (h *TeamsBetaHandler) GetChatMembers(c *gin.Context) {
	chatId := c.Param("chatId")
	result, err := h.graphService.GetBeta("/chats/" + chatId + "/members")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/beta/chats/:chatId/members/:membershipId
func (h *TeamsBetaHandler) GetChatMember(c *gin.Context) {
	chatId := c.Param("chatId")
	membershipId := c.Param("membershipId")
	result, err := h.graphService.GetBeta("/chats/" + chatId + "/members/" + membershipId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
