package handlers

import (
	"net/http"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type MailHandler struct {
	graphService *services.GraphService
}

func NewMailHandler(gs *services.GraphService) *MailHandler {
	return &MailHandler{graphService: gs}
}

// GET /api/mail/important
func (h *MailHandler) GetHighImportanceMail(c *gin.Context) {
	result, err := h.graphService.Get("/me/messages?$filter=importance eq 'high'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/mail/from/:email
func (h *MailHandler) GetMailFromAddress(c *gin.Context) {
	email := c.Param("email")
	result, err := h.graphService.Get("/me/messages?$filter=(from/emailAddress/address) eq '" + email + "'")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/mail/send
func (h *MailHandler) SendMail(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/me/sendMail", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, result)
}

// POST /api/mail/:messageId/forward
func (h *MailHandler) ForwardMail(c *gin.Context) {
	messageId := c.Param("messageId")
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/me/messages/"+messageId+"/forward", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, result)
}

// GET /api/mail/delta
func (h *MailHandler) GetMailDelta(c *gin.Context) {
	result, err := h.graphService.Get("/me/mailFolders/Inbox/messages/delta")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
