package services

import "fmt"

type TeamsChatService struct {
	graphService *GraphService
}

func NewTeamsChatService(gs *GraphService) *TeamsChatService {
	return &TeamsChatService{graphService: gs}
}

// Créer un chat 1:1 avec un utilisateur
func (s *TeamsChatService) CreateOneOnOneChat(userID string) (string, error) {
	body := map[string]any{
		"chatType": "oneOnOne",
		"members": []map[string]any{
			{
				"@odata.type":     "#microsoft.graph.aadUserConversationMember",
				"roles":           []string{"owner"},
				"user@odata.bind": fmt.Sprintf("https://graph.microsoft.com/v1.0/users('%s')", userID),
			},
		},
	}

	result, err := s.graphService.Post("/chats", body)
	if err != nil {
		return "", err
	}

	if chatID, ok := result["id"].(string); ok {
		return chatID, nil
	}
	return "", fmt.Errorf("no chat id in response")
}

// Envoyer un message dans un chat
func (s *TeamsChatService) SendChatMessage(chatID, message string) error {
	body := map[string]any{
		"body": map[string]any{
			"content": message,
		},
	}

	_, err := s.graphService.Post("/chats/"+chatID+"/messages", body)
	return err
}

// Envoyer un message à un utilisateur (crée le chat si nécessaire)
func (s *TeamsChatService) SendDirectMessage(userID, message string) error {
	chatID, err := s.CreateOneOnOneChat(userID)
	if err != nil {
		return err
	}
	return s.SendChatMessage(chatID, message)
}

// Envoyer un message dans un canal
func (s *TeamsChatService) SendChannelMessage(teamID, channelID, message string) error {
	body := map[string]any{
		"body": map[string]any{
			"content": message,
		},
	}

	endpoint := fmt.Sprintf("/teams/%s/channels/%s/messages", teamID, channelID)
	_, err := s.graphService.Post(endpoint, body)
	return err
}
