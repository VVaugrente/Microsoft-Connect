package handlers

import (
	"net/http"
	"time"

	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

type CalendarHandler struct {
	graphService *services.GraphService
}

func NewCalendarHandler(gs *services.GraphService) *CalendarHandler {
	return &CalendarHandler{graphService: gs}
}

// GET /api/calendar/week
func (h *CalendarHandler) GetNextWeekEvents(c *gin.Context) {
	now := time.Now().UTC()
	nextWeek := now.AddDate(0, 0, 7)
	startDate := now.Format(time.RFC3339)
	endDate := nextWeek.Format(time.RFC3339)

	result, err := h.graphService.Get("/me/calendarview?startdatetime=" + startDate + "&enddatetime=" + endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/calendar/events
func (h *CalendarHandler) GetAllEvents(c *gin.Context) {
	result, err := h.graphService.Get("/me/events?$select=subject,body,bodyPreview,organizer,attendees,start,end,location")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/calendar/calendars
func (h *CalendarHandler) GetAllCalendars(c *gin.Context) {
	result, err := h.graphService.Get("/me/calendars")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/calendar/findMeetingTimes
func (h *CalendarHandler) FindMeetingTimes(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/me/findMeetingTimes", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/calendar/events
func (h *CalendarHandler) CreateEvent(c *gin.Context) {
	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.graphService.Post("/me/events", body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}
