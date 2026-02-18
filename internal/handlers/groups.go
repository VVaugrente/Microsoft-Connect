package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type GroupsHandler struct {
	graphService *services.GraphService
}

func NewGroupsHandler(gs *services.GraphService) *GroupsHandler {
	return &GroupsHandler{graphService: gs}
}

// GET /api/groups
func (h *GroupsHandler) GetAllGroups(c *gin.Context) {
	result, err := h.graphService.Get("/groups")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/groups
func (h *GroupsHandler) CreateGroup(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/groups", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// POST /api/groups/:groupId/members
func (h *GroupsHandler) AddMember(c *gin.Context) {
	groupId := c.Param("groupId")
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/groups/"+groupId+"/members/$ref", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, result)
}

// DELETE /api/groups/:groupId/members/:memberId
func (h *GroupsHandler) RemoveMember(c *gin.Context) {
	groupId := c.Param("groupId")
	memberId := c.Param("memberId")
	err := h.graphService.Delete("/groups/" + groupId + "/members/" + memberId + "/$ref")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// DELETE /api/groups/:groupId
func (h *GroupsHandler) DeleteGroup(c *gin.Context) {
	groupId := c.Param("groupId")
	err := h.graphService.Delete("/groups/" + groupId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// GET /api/groups/me
func (h *GroupsHandler) GetMyGroups(c *gin.Context) {
	result, err := h.graphService.Get("/me/transitiveMemberOf/microsoft.graph.group?$count=true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/groups/:groupId/conversations
func (h *GroupsHandler) GetGroupConversations(c *gin.Context) {
	groupId := c.Param("groupId")
	result, err := h.graphService.Get("/groups/" + groupId + "/conversations")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/groups/:groupId/events
func (h *GroupsHandler) GetGroupEvents(c *gin.Context) {
	groupId := c.Param("groupId")
	result, err := h.graphService.Get("/groups/" + groupId + "/events")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
