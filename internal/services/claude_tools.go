package services

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

func GetMicrosoftTools() []Tool {
	return []Tool{
		{
			Name:        "get_calendar_events", // done
			Description: "Récupère les événements du calendrier d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID ou email de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "send_email", // done
			Description: "Envoie un email au nom d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"from":    map[string]string{"type": "string", "description": "Email de l'expéditeur"},
					"to":      map[string]string{"type": "string", "description": "Adresse email du destinataire"},
					"subject": map[string]string{"type": "string", "description": "Sujet de l'email"},
					"body":    map[string]string{"type": "string", "description": "Contenu de l'email"},
				},
				"required": []string{"from", "to", "subject", "body"},
			},
		},
		{
			Name:        "get_users", // done
			Description: "Liste les utilisateurs de l'organisation",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_teams",
			Description: "Liste les équipes Teams de l'utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID de l'utilisateur pour récupérer ses équipes Teams"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "create_meeting", // done
			Description: "Créer une réunion Teams pour un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id":    map[string]string{"type": "string", "description": "ID ou email de l'organisateur"},
					"subject":    map[string]string{"type": "string", "description": "Sujet de la réunion"},
					"start_time": map[string]string{"type": "string", "description": "Date/heure de début (ISO 8601)"},
					"end_time":   map[string]string{"type": "string", "description": "Date/heure de fin (ISO 8601)"},
					"attendees":  map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Liste des emails des participants"},
				},
				"required": []string{"user_id", "subject", "start_time", "end_time"},
			},
		},
		{
			Name:        "find_meeting_times",
			Description: "Trouver des créneaux disponibles pour une réunion",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"attendees":        map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Emails des participants"},
					"duration_minutes": map[string]string{"type": "integer", "description": "Durée de la réunion en minutes"},
				},
				"required": []string{"attendees", "duration_minutes"},
			},
		},
		{
			Name:        "send_channel_message", // done
			Description: "Envoyer un message dans un canal Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id":    map[string]string{"type": "string", "description": "ID de l'équipe"},
					"channel_id": map[string]string{"type": "string", "description": "ID du canal"},
					"message":    map[string]string{"type": "string", "description": "Contenu du message"},
				},
				"required": []string{"team_id", "channel_id", "message"},
			},
		},
		{
			Name:        "get_user_presence",
			Description: "Obtenir le statut de présence d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID ou email de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},
	}
}
