package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type TeamsHandler struct {
	graphService *services.GraphService
}

func NewTeamsHandler(gs *services.GraphService) *TeamsHandler {
	return &TeamsHandler{graphService: gs}
}

// POST /api/teams
func (h *TeamsHandler) CreateTeam(c *gin.Context) {
	var body map[string]any
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/teams", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// GET /api/teams/joined
func (h *TeamsHandler) GetJoinedTeams(c *gin.Context) {
	result, err := h.graphService.Get("/me/joinedTeams")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/teams/:id/members
func (h *TeamsHandler) GetTeamMembers(c *gin.Context) {
	id := c.Param("id")
	result, err := h.graphService.Get("/groups/" + id + "/members")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/teams/:id/channels
func (h *TeamsHandler) GetTeamChannels(c *gin.Context) {
	id := c.Param("id")
	result, err := h.graphService.Get("/teams/" + id + "/channels")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/teams/:id/channels/:channelId
func (h *TeamsHandler) GetChannelInfo(c *gin.Context) {
	id := c.Param("id")
	channelId := c.Param("channelId")
	result, err := h.graphService.Get("/teams/" + id + "/channels/" + channelId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/teams/:id/channels
func (h *TeamsHandler) CreateChannel(c *gin.Context) {
	id := c.Param("id")
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/teams/"+id+"/channels", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// GET /api/teams/:id/apps
func (h *TeamsHandler) GetTeamApps(c *gin.Context) {
	id := c.Param("id")
	result, err := h.graphService.Get("/teams/" + id + "/installedApps?$expand=teamsAppDefinition")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/chats
func (h *TeamsHandler) CreateChat(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/chats", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// POST /api/teams/:id/channels/:channelId/messages
func (h *TeamsHandler) SendChannelMessage(c *gin.Context) {
	id := c.Param("id")
	channelId := c.Param("channelId")
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/teams/"+id+"/channels/"+channelId+"/messages", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}
