package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type ClaudeService struct {
	apiKey            string
	httpClient        *http.Client
	conversationStore *ConversationStore
}

type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type ClaudeResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type ToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

func NewClaudeService(apiKey string) *ClaudeService {
	return &ClaudeService{
		apiKey:            apiKey,
		httpClient:        &http.Client{},
		conversationStore: NewConversationStore(),
	}
}

func (s *ClaudeService) SendMessage(userMessage string, context string) (string, error) {
	reqBody := ClaudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "user", Content: context + "\n\n" + userMessage},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result ClaudeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	return "", fmt.Errorf("empty response from Claude")
}

func (s *ClaudeService) SendMessageWithContext(userMessage string, context string, conversationID string, graphService *GraphService) (string, error) {
	// Récupérer l'historique
	history := s.conversationStore.GetHistory(conversationID)

	// Construire les messages avec l'historique
	messages := []Message{}

	// Ajouter le contexte système comme premier message utilisateur si pas d'historique
	if len(history) == 0 {
		messages = append(messages, Message{Role: "user", Content: context + "\n\n" + userMessage})
	} else {
		// Reconstruire l'historique
		for i, msg := range history {
			if i == 0 {
				// Premier message avec contexte
				messages = append(messages, Message{Role: msg.Role, Content: context + "\n\n" + msg.Content})
			} else {
				messages = append(messages, Message{Role: msg.Role, Content: msg.Content})
			}
		}
		// Ajouter le nouveau message
		messages = append(messages, Message{Role: "user", Content: userMessage})
	}

	// Sauvegarder le message utilisateur
	s.conversationStore.AddMessage(conversationID, "user", userMessage)

	response, err := s.sendWithTools(messages, graphService)
	if err != nil {
		return "", err
	}

	// Sauvegarder la réponse
	s.conversationStore.AddMessage(conversationID, "assistant", response)

	return response, nil
}

func (s *ClaudeService) SendMessageWithTools(userMessage string, context string, graphService *GraphService) (string, error) {
	messages := []Message{
		{Role: "user", Content: context + "\n\n" + userMessage},
	}
	return s.sendWithTools(messages, graphService)
}

func (s *ClaudeService) sendWithTools(messages []Message, graphService *GraphService) (string, error) {
	for {
		reqBody := ClaudeRequest{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			Messages:  messages,
			Tools:     GetMicrosoftTools(),
		}

		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result ClaudeResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		if result.StopReason == "end_turn" {
			for _, block := range result.Content {
				if block.Type == "text" {
					return block.Text, nil
				}
			}
			return "", fmt.Errorf("no text in response")
		}

		if result.StopReason == "tool_use" {
			messages = append(messages, Message{Role: "assistant", Content: result.Content})

			var toolResults []ToolResult
			for _, block := range result.Content {
				if block.Type == "tool_use" {
					toolResult := s.executeTool(block.Name, block.Input, graphService)
					toolResults = append(toolResults, ToolResult{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   toolResult,
					})
				}
			}

			messages = append(messages, Message{Role: "user", Content: toolResults})
		}
	}
}

func (s *ClaudeService) executeTool(toolName string, input json.RawMessage, graphService *GraphService) string {
	var result map[string]any
	var err error

	log.Printf("=== EXECUTING TOOL: %s ===", toolName)
	log.Printf("Input: %s", string(input))

	switch toolName {
	// === CALENDRIER ===
	case "get_calendar_events":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/events?$select=subject,start,end,location&$orderby=start/dateTime&$top=10")

	case "get_calendars":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/calendars")

	case "create_meeting":
		var params struct {
			UserID    string   `json:"user_id"`
			Subject   string   `json:"subject"`
			StartTime string   `json:"start_time"`
			EndTime   string   `json:"end_time"`
			Attendees []string `json:"attendees"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" || params.Subject == "" || params.StartTime == "" || params.EndTime == "" {
			return "Erreur: user_id, subject, start_time et end_time requis"
		}

		attendeesList := []map[string]any{}
		for _, email := range params.Attendees {
			attendeesList = append(attendeesList, map[string]any{
				"emailAddress": map[string]string{"address": email},
				"type":         "required",
			})
		}

		body := map[string]any{
			"subject": params.Subject,
			"start": map[string]string{
				"dateTime": params.StartTime,
				"timeZone": "Europe/Paris",
			},
			"end": map[string]string{
				"dateTime": params.EndTime,
				"timeZone": "Europe/Paris",
			},
			"attendees":           attendeesList,
			"isOnlineMeeting":     true,
			"onlineMeetingProvider": "teamsForBusiness",
		}
		result, err = graphService.Post("/users/"+params.UserID+"/events", body)

	case "find_meeting_times":
		var params struct {
			Attendees       []string `json:"attendees"`
			DurationMinutes int      `json:"duration_minutes"`
		}
		json.Unmarshal(input, &params)
		if len(params.Attendees) == 0 || params.DurationMinutes == 0 {
			return "Erreur: attendees et duration_minutes requis"
		}

		attendeesList := []map[string]any{}
		for _, email := range params.Attendees {
			attendeesList = append(attendeesList, map[string]any{
				"emailAddress": map[string]string{"address": email},
				"type":         "required",
			})
		}

		body := map[string]any{
			"attendees": attendeesList,
			"timeConstraint": map[string]any{
				"timeslots": []map[string]any{
					{
						"start": map[string]string{
							"dateTime": time.Now().Format("2006-01-02T09:00:00"),
							"timeZone": "Europe/Paris",
						},
						"end": map[string]string{
							"dateTime": time.Now().AddDate(0, 0, 7).Format("2006-01-02T18:00:00"),
							"timeZone": "Europe/Paris",
						},
					},
				},
			},
			"meetingDuration": fmt.Sprintf("PT%dM", params.DurationMinutes),
		}
		result, err = graphService.Post("/me/findMeetingTimes", body)

	// === MESSAGERIE OUTLOOK ===
	case "send_email":
		var params struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Subject string `json:"subject"`
			Body    string `json:"body"`
		}
		json.Unmarshal(input, &params)
		if params.From == "" || params.To == "" || params.Subject == "" || params.Body == "" {
			return "Erreur: from, to, subject et body requis"
		}

		body := map[string]any{
			"message": map[string]any{
				"subject": params.Subject,
				"body": map[string]any{
					"contentType": "Text",
					"content":     params.Body,
				},
				"toRecipients": []map[string]any{
					{"emailAddress": map[string]string{"address": params.To}},
				},
			},
		}
		result, err = graphService.Post("/users/"+params.From+"/sendMail", body)
		if err == nil {
			return fmt.Sprintf("Email envoyé à %s avec succès", params.To)
		}

	case "get_important_emails":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/messages?$filter=importance eq 'high'&$top=20")

	case "get_emails_from":
		var params struct {
			UserID    string `json:"user_id"`
			FromEmail string `json:"from_email"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" || params.FromEmail == "" {
			return "Erreur: user_id et from_email requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/messages?$filter=(from/emailAddress/address) eq '" + params.FromEmail + "'&$top=20")

	case "forward_email":
		var params struct {
			UserID    string `json:"user_id"`
			MessageID string `json:"message_id"`
			ToEmail   string `json:"to_email"`
			Comment   string `json:"comment"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" || params.MessageID == "" || params.ToEmail == "" {
			return "Erreur: user_id, message_id et to_email requis"
		}

		body := map[string]any{
			"comment": params.Comment,
			"toRecipients": []map[string]any{
				{"emailAddress": map[string]string{"address": params.ToEmail}},
			},
		}
		result, err = graphService.Post("/users/"+params.UserID+"/messages/"+params.MessageID+"/forward", body)
		if err == nil {
			return "Email transféré avec succès"
		}

	case "get_email_delta":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/mailFolders/Inbox/messages/delta")

	// === UTILISATEURS ===
	case "get_users":
		result, err = graphService.Get("/users?$select=id,displayName,mail,userPrincipalName&$top=50")

	case "get_user_presence":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/presence")

	// === TEAMS ===
	case "get_teams":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/joinedTeams")

	case "create_team":
		var params struct {
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			Visibility  string `json:"visibility"`
		}
		json.Unmarshal(input, &params)
		if params.DisplayName == "" {
			return "Erreur: display_name requis"
		}

		visibility := "private"
		if params.Visibility != "" {
			visibility = params.Visibility
		}

		body := map[string]any{
			"template@odata.bind": "https://graph.microsoft.com/v1.0/teamsTemplates('standard')",
			"displayName":         params.DisplayName,
			"description":         params.Description,
			"visibility":          visibility,
		}
		result, err = graphService.Post("/teams", body)
		if err == nil {
			return fmt.Sprintf("Équipe '%s' créée avec succès", params.DisplayName)
		}

	case "get_team_members":
		var params struct {
			TeamID string `json:"team_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" {
			return "Erreur: team_id requis"
		}
		result, err = graphService.Get("/groups/" + params.TeamID + "/members")

	case "get_team_channels":
		var params struct {
			TeamID string `json:"team_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" {
			return "Erreur: team_id requis"
		}
		result, err = graphService.Get("/teams/" + params.TeamID + "/channels")

	case "get_channel_info":
		var params struct {
			TeamID    string `json:"team_id"`
			ChannelID string `json:"channel_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" || params.ChannelID == "" {
			return "Erreur: team_id et channel_id requis"
		}
		result, err = graphService.Get("/teams/" + params.TeamID + "/channels/" + params.ChannelID)

	case "create_channel":
		var params struct {
			TeamID         string `json:"team_id"`
			DisplayName    string `json:"display_name"`
			Description    string `json:"description"`
			MembershipType string `json:"membership_type"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" || params.DisplayName == "" {
			return "Erreur: team_id et display_name requis"
		}

		membershipType := "standard"
		if params.MembershipType != "" {
			membershipType = params.MembershipType
		}

		body := map[string]any{
			"displayName":    params.DisplayName,
			"description":    params.Description,
			"membershipType": membershipType,
		}
		result, err = graphService.Post("/teams/"+params.TeamID+"/channels", body)
		if err == nil {
			return fmt.Sprintf("Canal '%s' créé avec succès", params.DisplayName)
		}

	case "get_team_apps":
		var params struct {
			TeamID string `json:"team_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" {
			return "Erreur: team_id requis"
		}
		result, err = graphService.Get("/teams/" + params.TeamID + "/installedApps?$expand=teamsAppDefinition")

	case "create_chat":
		var params struct {
			ChatType string   `json:"chat_type"`
			Members  []string `json:"members"`
			Topic    string   `json:"topic"`
		}
		json.Unmarshal(input, &params)
		if params.ChatType == "" || len(params.Members) == 0 {
			return "Erreur: chat_type et members requis"
		}

		membersList := []map[string]any{}
		for _, userID := range params.Members {
			membersList = append(membersList, map[string]any{
				"@odata.type":     "#microsoft.graph.aadUserConversationMember",
				"roles":           []string{"owner"},
				"user@odata.bind": fmt.Sprintf("https://graph.microsoft.com/v1.0/users('%s')", userID),
			})
		}

		body := map[string]any{
			"chatType": params.ChatType,
			"members":  membersList,
		}
		if params.Topic != "" {
			body["topic"] = params.Topic
		}
		result, err = graphService.Post("/chats", body)

	case "send_chat_message":
		var params struct {
			ChatID  string `json:"chat_id"`
			Message string `json:"message"`
		}
		json.Unmarshal(input, &params)
		if params.ChatID == "" || params.Message == "" {
			return "Erreur: chat_id et message requis"
		}

		body := map[string]any{
			"body": map[string]any{
				"content": params.Message,
			},
		}
		result, err = graphService.Post("/chats/"+params.ChatID+"/messages", body)
		if err == nil {
			return "Message envoyé avec succès"
		}

	case "send_channel_message":
		var params struct {
			TeamID    string `json:"team_id"`
			ChannelID string `json:"channel_id"`
			Message   string `json:"message"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" || params.ChannelID == "" || params.Message == "" {
			return "Erreur: team_id, channel_id et message requis"
		}

		body := map[string]any{
			"body": map[string]any{
				"content": params.Message,
			},
		}
		result, err = graphService.Post("/teams/"+params.TeamID+"/channels/"+params.ChannelID+"/messages", body)
		if err == nil {
			return "Message envoyé dans le canal avec succès"
		}

	// === GROUPES ===
	case "get_groups":
		result, err = graphService.Get("/groups")

	case "create_group":
		var params struct {
			DisplayName     string `json:"display_name"`
			Description     string `json:"description"`
			MailNickname    string `json:"mail_nickname"`
			MailEnabled     bool   `json:"mail_enabled"`
			SecurityEnabled bool   `json:"security_enabled"`
		}
		json.Unmarshal(input, &params)
		if params.DisplayName == "" || params.MailNickname == "" {
			return "Erreur: display_name et mail_nickname requis"
		}

		groupBody := map[string]any{
			"displayName":     params.DisplayName,
			"description":     params.Description,
			"mailNickname":    params.MailNickname,
			"mailEnabled":     params.MailEnabled,
			"securityEnabled": params.SecurityEnabled,
			"groupTypes":      []string{"Unified"},
		}
		result, err = graphService.Post("/groups", groupBody)
		if err == nil {
			return fmt.Sprintf("Groupe '%s' créé avec succès", params.DisplayName)
		}

	case "add_group_member":
		var params struct {
			GroupID string `json:"group_id"`
			UserID  string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.GroupID == "" || params.UserID == "" {
			return "Erreur: group_id et user_id requis"
		}

		body := map[string]any{
			"@odata.id": fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s", params.UserID),
		}
		result, err = graphService.Post("/groups/"+params.GroupID+"/members/$ref", body)
		if err == nil {
			return "Membre ajouté au groupe avec succès"
		}

	case "remove_group_member":
		var params struct {
			GroupID string `json:"group_id"`
			UserID  string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.GroupID == "" || params.UserID == "" {
			return "Erreur: group_id et user_id requis"
		}

		err = graphService.Delete("/groups/" + params.GroupID + "/members/" + params.UserID + "/$ref")
		if err == nil {
			return "Membre supprimé du groupe avec succès"
		}

	case "delete_group":
		var params struct {
			GroupID string `json:"group_id"`
		}
		json.Unmarshal(input, &params)
		if params.GroupID == "" {
			return "Erreur: group_id requis"
		}

		err = graphService.Delete("/groups/" + params.GroupID)
		if err == nil {
			return "Groupe supprimé avec succès"
		}

	case "get_my_groups":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.Get("/users/" + params.UserID + "/transitiveMemberOf/microsoft.graph.group?$count=true")

	case "get_group_conversations":
		var params struct {
			GroupID string `json:"group_id"`
		}
		json.Unmarshal(input, &params)
		if params.GroupID == "" {
			return "Erreur: group_id requis"
		}
		result, err = graphService.Get("/groups/" + params.GroupID + "/conversations")

	case "get_group_events":
		var params struct {
			GroupID string `json:"group_id"`
		}
		json.Unmarshal(input, &params)
		if params.GroupID == "" {
			return "Erreur: group_id requis"
		}
		result, err = graphService.Get("/groups/" + params.GroupID + "/events")

	// === TEAMS BETA ===
	case "get_channel_messages":
		var params struct {
			TeamID    string `json:"team_id"`
			ChannelID string `json:"channel_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" || params.ChannelID == "" {
			return "Erreur: team_id et channel_id requis"
		}
		result, err = graphService.GetBeta("/teams/" + params.TeamID + "/channels/" + params.ChannelID + "/messages")

	case "get_message_replies":
		var params struct {
			TeamID    string `json:"team_id"`
			ChannelID string `json:"channel_id"`
			MessageID string `json:"message_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" || params.ChannelID == "" || params.MessageID == "" {
			return "Erreur: team_id, channel_id et message_id requis"
		}
		result, err = graphService.GetBeta("/teams/" + params.TeamID + "/channels/" + params.ChannelID + "/messages/" + params.MessageID + "/replies")

	case "get_installed_apps":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}
		result, err = graphService.GetBeta("/users/" + params.UserID + "/teamwork/installedApps?$expand=teamsApp")

	case "get_chat_members":
		var params struct {
			ChatID string `json:"chat_id"`
		}
		json.Unmarshal(input, &params)
		if params.ChatID == "" {
			return "Erreur: chat_id requis"
		}
		result, err = graphService.GetBeta("/chats/" + params.ChatID + "/members")

	default:
		return fmt.Sprintf("Outil inconnu: %s", toolName)
	}

	if err != nil {
		return fmt.Sprintf("Erreur: %s", err.Error())
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult)
}
