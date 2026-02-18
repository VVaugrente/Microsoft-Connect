package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"microsoft_connector/config"
	"microsoft_connector/internal/handlers"
	"microsoft_connector/internal/services"

	"github.com/gin-gonic/gin"
)

// Self-ping pour éviter le cold start de Render
func startSelfPing(url string) {
	go func() {
		// Attendre que le serveur démarre
		time.Sleep(10 * time.Second)

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		client := &http.Client{Timeout: 10 * time.Second}

		for {
			resp, err := client.Get(url + "/health")
			if err != nil {
				log.Printf("Self-ping failed: %v", err)
			} else {
				resp.Body.Close()
				log.Printf("Self-ping OK")
			}
			<-ticker.C
		}
	}()
}

func main() {
	cfg := config.Load()

	port := cfg.Port
	if port == "" {
		port = "10000"
	}
	log.Printf("Starting server on port %s", port)

	// Démarrer le self-ping si on est sur Render
	renderURL := os.Getenv("RENDER_EXTERNAL_URL")
	if renderURL != "" {
		log.Printf("Render detected, starting self-ping to %s", renderURL)
		startSelfPing(renderURL)
	}

	// Services
	authService := services.NewAuthService(cfg)
	graphService := services.NewGraphService(authService)
	claudeService := services.NewClaudeService(os.Getenv("CLAUDE_API_KEY"))

	// Handlers
	usersHandler := handlers.NewUsersHandler(graphService)
	teamsHandler := handlers.NewTeamsHandler(graphService)
	teamsBetaHandler := handlers.NewTeamsBetaHandler(graphService)
	groupsHandler := handlers.NewGroupsHandler(graphService)
	calendarHandler := handlers.NewCalendarHandler(graphService)
	mailHandler := handlers.NewMailHandler(graphService)
	batchHandler := handlers.NewBatchHandler(graphService)
	chatHandler := handlers.NewChatHandler(claudeService, graphService)
	teamsBotHandler := handlers.NewTeamsBotHandler(claudeService, graphService, os.Getenv("TEAMS_WEBHOOK_SECRET"))

	r := gin.Default()

	// Route racine
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "microsoft-connector"})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		// Webhook Teams
		api.GET("/webhook", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ready"})
		})
		api.HEAD("/webhook", func(c *gin.Context) {
			c.Status(200)
		})
		api.POST("/webhook", teamsBotHandler.HandleWebhook)

		// Users
		users := api.Group("/users")
		{
			users.GET("", usersHandler.GetAllUsers)
			users.GET("/guests", usersHandler.GetGuestUsers)
			users.GET("/:email", usersHandler.GetUserByEmail)
		}

		// Teams
		teams := api.Group("/teams")
		{
			teams.POST("", teamsHandler.CreateTeam)
			teams.GET("/joined", teamsHandler.GetJoinedTeams)
			teams.GET("/:id/members", teamsHandler.GetTeamMembers)
			teams.GET("/:id/channels", teamsHandler.GetTeamChannels)
			teams.GET("/:id/channels/:channelId", teamsHandler.GetChannelInfo)
			teams.POST("/:id/channels", teamsHandler.CreateChannel)
			teams.GET("/:id/apps", teamsHandler.GetTeamApps)
			teams.POST("/:id/channels/:channelId/messages", teamsHandler.SendChannelMessage)
		}

		// Chats
		api.POST("/chats", teamsHandler.CreateChat)

		// Teams Beta
		beta := api.Group("/beta")
		{
			beta.GET("/teams/:id/channels/:channelId/messages", teamsBetaHandler.GetChannelMessages)
			beta.GET("/teams/:id/channels/:channelId/messages/:messageId/replies", teamsBetaHandler.GetMessageReplies)
			beta.GET("/me/apps", teamsBetaHandler.GetMyInstalledApps)
			beta.GET("/chats/:chatId/members", teamsBetaHandler.GetChatMembers)
		}

		// Groups
		groups := api.Group("/groups")
		{
			groups.GET("", groupsHandler.GetAllGroups)
			groups.POST("", groupsHandler.CreateGroup)
			groups.POST("/:groupId/members", groupsHandler.AddMember)
			groups.DELETE("/:groupId/members/:memberId", groupsHandler.RemoveMember)
			groups.DELETE("/:groupId", groupsHandler.DeleteGroup)
		}

		// Calendar
		calendar := api.Group("/calendar")
		{
			calendar.GET("/week", calendarHandler.GetNextWeekEvents)
			calendar.GET("/events", calendarHandler.GetAllEvents)
			calendar.GET("/calendars", calendarHandler.GetAllCalendars)
			calendar.POST("/findMeetingTimes", calendarHandler.FindMeetingTimes)
			calendar.POST("/events", calendarHandler.CreateEvent)
		}

		// Mail
		mail := api.Group("/mail")
		{
			mail.GET("/important", mailHandler.GetHighImportanceMail)
			mail.GET("/from/:email", mailHandler.GetMailFromAddress)
			mail.POST("/send", mailHandler.SendMail)
			mail.POST("/:messageId/forward", mailHandler.ForwardMail)
		}

		// Chat avec IA
		chat := api.Group("/chat")
		{
			chat.POST("", chatHandler.Chat)
			chat.POST("/direct", chatHandler.SendDirectMessage)
			chat.POST("/channel", chatHandler.SendChannelMessage)
		}

		// Batch
		api.POST("/batch", batchHandler.ExecuteBatch)
	}

	addr := "0.0.0.0:" + port
	log.Printf("Server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
