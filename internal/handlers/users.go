package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type UsersHandler struct {
	graphService *services.GraphService
}

func NewUsersHandler(gs *services.GraphService) *UsersHandler {
	return &UsersHandler{graphService: gs}
}

// GET /api/users/me
func (h *UsersHandler) GetMyProfile(c *gin.Context) {
	result, err := h.graphService.Get("/me")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/users
func (h *UsersHandler) GetAllUsers(c *gin.Context) {
	result, err := h.graphService.Get("/users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/users/guests
func (h *UsersHandler) GetGuestUsers(c *gin.Context) {
	result, err := h.graphService.Get("/users/?$filter=userType eq 'guest'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/users/:email
func (h *UsersHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")
	result, err := h.graphService.Get("/users/" + email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
