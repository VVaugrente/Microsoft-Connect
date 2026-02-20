package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	case "get_calendar_events":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		log.Printf("Calendar request for user_id: %s", params.UserID)
		if params.UserID == "" {
			return "Erreur: user_id requis pour accéder au calendrier. Utilise l'ID utilisateur fourni dans le contexte."
		}
		endpoint := "/users/" + params.UserID + "/events?$select=subject,start,end,location,onlineMeeting&$top=10&$orderby=start/dateTime"
		log.Printf("Graph endpoint: %s", endpoint)
		result, err = graphService.Get(endpoint)

		// LOG DÉTAILLÉ DU RÉSULTAT
		if err != nil {
			log.Printf("=== CALENDAR ERROR ===")
			log.Printf("Error: %v", err)
			return fmt.Sprintf("Erreur lors de la récupération du calendrier: %s", err.Error())
		}

		jsonResult, _ := json.MarshalIndent(result, "", "  ")
		log.Printf("=== CALENDAR RESULT ===")
		log.Printf("%s", string(jsonResult))

		// Compter les événements
		if value, ok := result["value"].([]interface{}); ok {
			log.Printf("Nombre d'événements trouvés: %d", len(value))
		}

		return string(jsonResult)

	case "get_users":
		result, err = graphService.Get("/users?$select=displayName,mail,jobTitle,id&$top=20")

	case "get_teams":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis pour lister les équipes Teams. Utilise l'ID utilisateur fourni dans le contexte."
		}
		result, err = graphService.Get("/users/" + params.UserID + "/joinedTeams")

	case "send_email":
		var params struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Subject string `json:"subject"`
			Body    string `json:"body"`
		}
		json.Unmarshal(input, &params)
		if params.From == "" {
			return "Erreur: from (email expéditeur) requis"
		}

		emailBody := map[string]any{
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
		result, err = graphService.Post("/users/"+params.From+"/sendMail", emailBody)
		if err == nil {
			return "Email envoyé avec succès"
		}

	case "create_meeting":
		var params struct {
			UserID    string   `json:"user_id"`
			Subject   string   `json:"subject"`
			StartTime string   `json:"start_time"`
			EndTime   string   `json:"end_time"`
			Attendees []string `json:"attendees"`
		}
		json.Unmarshal(input, &params)
		if params.UserID == "" {
			return "Erreur: user_id requis"
		}

		attendees := make([]map[string]any, len(params.Attendees))
		for i, email := range params.Attendees {
			attendees[i] = map[string]any{
				"emailAddress": map[string]string{"address": email},
				"type":         "required",
			}
		}

		meetingBody := map[string]any{
			"subject": params.Subject,
			"start": map[string]string{
				"dateTime": params.StartTime,
				"timeZone": "Europe/Paris",
			},
			"end": map[string]string{
				"dateTime": params.EndTime,
				"timeZone": "Europe/Paris",
			},
			"attendees":             attendees,
			"isOnlineMeeting":       true,
			"onlineMeetingProvider": "teamsForBusiness",
		}
		result, err = graphService.Post("/users/"+params.UserID+"/events", meetingBody)
		if err == nil {
			if onlineMeeting, ok := result["onlineMeeting"].(map[string]any); ok {
				if joinURL, ok := onlineMeeting["joinUrl"].(string); ok {
					return fmt.Sprintf("Réunion créée ! Lien: %s", joinURL)
				}
			}
			return "Réunion créée avec succès"
		}

	case "find_meeting_times":
		var params struct {
			Attendees       []string `json:"attendees"`
			DurationMinutes int      `json:"duration_minutes"`
		}
		json.Unmarshal(input, &params)

		attendees := make([]map[string]any, len(params.Attendees))
		for i, email := range params.Attendees {
			attendees[i] = map[string]any{
				"emailAddress": map[string]string{"address": email},
				"type":         "required",
			}
		}

		body := map[string]any{
			"attendees":       attendees,
			"meetingDuration": fmt.Sprintf("PT%dM", params.DurationMinutes),
		}
		result, err = graphService.Post("/me/findMeetingTimes", body)

	case "send_channel_message":
		var params struct {
			TeamID    string `json:"team_id"`
			ChannelID string `json:"channel_id"`
			Message   string `json:"message"`
		}
		json.Unmarshal(input, &params)

		body := map[string]any{
			"body": map[string]any{
				"content": params.Message,
			},
		}
		endpoint := fmt.Sprintf("/teams/%s/channels/%s/messages", params.TeamID, params.ChannelID)
		result, err = graphService.Post(endpoint, body)
		if err == nil {
			return "Message envoyé dans le canal"
		}

	case "get_user_presence":
		var params struct {
			UserID string `json:"user_id"`
		}
		json.Unmarshal(input, &params)
		result, err = graphService.GetBeta("/users/" + params.UserID + "/presence")

	case "get_team_channels":
		var params struct {
			TeamID string `json:"team_id"`
		}
		json.Unmarshal(input, &params)
		if params.TeamID == "" {
			return "Erreur: team_id requis"
		}
		result, err = graphService.Get("/teams/" + params.TeamID + "/channels")

	default:
		return fmt.Sprintf("Outil inconnu: %s", toolName)
	}

	if err != nil {
		return fmt.Sprintf("Erreur: %s", err.Error())
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult)
}
