package services

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

func GetMicrosoftTools() []Tool {
	return []Tool{
		{
			Name:        "get_calendar_events",
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
			Name:        "send_email",
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
			Name:        "get_users",
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
			Name:        "create_team",
			Description: "Crée une nouvelle équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"display_name": map[string]string{"type": "string", "description": "Nom de l'équipe"},
					"description":  map[string]string{"type": "string", "description": "Description de l'équipe"},
					"visibility":   map[string]string{"type": "string", "description": "Visibilité: 'private' ou 'public'"},
				},
				"required": []string{"display_name"},
			},
		},
		{
			Name:        "get_team_members",
			Description: "Liste les membres d'une équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]string{"type": "string", "description": "ID de l'équipe Teams (aussi appelé group-id)"},
				},
				"required": []string{"team_id"},
			},
		},
		{
			Name:        "get_team_channels",
			Description: "Liste les canaux d'une équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]string{"type": "string", "description": "ID de l'équipe Teams"},
				},
				"required": []string{"team_id"},
			},
		},
		{
			Name:        "get_channel_info",
			Description: "Récupère les informations d'un canal Teams spécifique",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id":    map[string]string{"type": "string", "description": "ID de l'équipe Teams"},
					"channel_id": map[string]string{"type": "string", "description": "ID du canal"},
				},
				"required": []string{"team_id", "channel_id"},
			},
		},
		{
			Name:        "create_channel",
			Description: "Crée un nouveau canal dans une équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id":         map[string]string{"type": "string", "description": "ID de l'équipe Teams"},
					"display_name":    map[string]string{"type": "string", "description": "Nom du canal"},
					"description":     map[string]string{"type": "string", "description": "Description du canal"},
					"membership_type": map[string]string{"type": "string", "description": "Type: 'standard' ou 'private'"},
				},
				"required": []string{"team_id", "display_name"},
			},
		},
		{
			Name:        "get_team_apps",
			Description: "Liste les applications installées dans une équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]string{"type": "string", "description": "ID de l'équipe Teams"},
				},
				"required": []string{"team_id"},
			},
		},
		{
			Name:        "create_chat",
			Description: "Crée une nouvelle conversation (chat 1:1 ou groupe)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_type": map[string]string{"type": "string", "description": "Type de chat: 'oneOnOne' ou 'group'"},
					"members":   map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "Liste des IDs utilisateurs à ajouter au chat"},
					"topic":     map[string]string{"type": "string", "description": "Sujet du chat (pour les chats de groupe)"},
				},
				"required": []string{"chat_type", "members"},
			},
		},
		{
			Name:        "send_chat_message",
			Description: "Envoie un message dans un chat existant",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_id": map[string]string{"type": "string", "description": "ID du chat"},
					"message": map[string]string{"type": "string", "description": "Contenu du message"},
				},
				"required": []string{"chat_id", "message"},
			},
		},
		{
			Name:        "create_meeting",
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
			Name:        "send_channel_message",
			Description: "Envoyer un message dans un canal Teams (Note: nécessite des permissions déléguées, peut échouer en mode application)",
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
		// === GROUPES ===
		{
			Name:        "get_groups",
			Description: "Liste tous les groupes de l'organisation",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "create_group",
			Description: "Crée un nouveau groupe Microsoft 365",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"display_name":     map[string]string{"type": "string", "description": "Nom du groupe"},
					"description":      map[string]string{"type": "string", "description": "Description du groupe"},
					"mail_nickname":    map[string]string{"type": "string", "description": "Alias email du groupe"},
					"mail_enabled":     map[string]string{"type": "boolean", "description": "Activer l'email"},
					"security_enabled": map[string]string{"type": "boolean", "description": "Groupe de sécurité"},
				},
				"required": []string{"display_name", "mail_nickname"},
			},
		},
		{
			Name:        "add_group_member",
			Description: "Ajoute un membre à un groupe",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_id": map[string]string{"type": "string", "description": "ID du groupe"},
					"user_id":  map[string]string{"type": "string", "description": "ID de l'utilisateur à ajouter"},
				},
				"required": []string{"group_id", "user_id"},
			},
		},
		{
			Name:        "remove_group_member",
			Description: "Supprime un membre d'un groupe",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_id": map[string]string{"type": "string", "description": "ID du groupe"},
					"user_id":  map[string]string{"type": "string", "description": "ID de l'utilisateur à supprimer"},
				},
				"required": []string{"group_id", "user_id"},
			},
		},
		{
			Name:        "delete_group",
			Description: "Supprime un groupe",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_id": map[string]string{"type": "string", "description": "ID du groupe à supprimer"},
				},
				"required": []string{"group_id"},
			},
		},
		{
			Name:        "get_my_groups",
			Description: "Liste les groupes auxquels l'utilisateur appartient",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_group_conversations",
			Description: "Récupère les conversations d'un groupe",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_id": map[string]string{"type": "string", "description": "ID du groupe"},
				},
				"required": []string{"group_id"},
			},
		},
		{
			Name:        "get_group_events",
			Description: "Récupère les événements d'un groupe",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_id": map[string]string{"type": "string", "description": "ID du groupe"},
				},
				"required": []string{"group_id"},
			},
		},

		// === MESSAGERIE OUTLOOK ===
		{
			Name:        "get_important_emails",
			Description: "Récupère les emails marqués comme importance haute",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_emails_from",
			Description: "Récupère les emails provenant d'une adresse spécifique",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id":    map[string]string{"type": "string", "description": "ID de l'utilisateur"},
					"from_email": map[string]string{"type": "string", "description": "Adresse email de l'expéditeur"},
				},
				"required": []string{"user_id", "from_email"},
			},
		},
		{
			Name:        "forward_email",
			Description: "Transfère un email à un destinataire",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id":    map[string]string{"type": "string", "description": "ID de l'utilisateur"},
					"message_id": map[string]string{"type": "string", "description": "ID du message à transférer"},
					"to_email":   map[string]string{"type": "string", "description": "Email du destinataire"},
					"comment":    map[string]string{"type": "string", "description": "Commentaire optionnel"},
				},
				"required": []string{"user_id", "message_id", "to_email"},
			},
		},
		{
			Name:        "get_email_delta",
			Description: "Récupère les modifications récentes des emails (nouveaux, modifiés)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},

		// === CALENDRIER ===
		{
			Name:        "get_calendars",
			Description: "Liste tous les calendriers de l'utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},

		// === TEAMS BETA ===
		{
			Name:        "get_channel_messages",
			Description: "Récupère les messages d'un canal Teams (sans les réponses)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id":    map[string]string{"type": "string", "description": "ID de l'équipe"},
					"channel_id": map[string]string{"type": "string", "description": "ID du canal"},
				},
				"required": []string{"team_id", "channel_id"},
			},
		},
		{
			Name:        "get_message_replies",
			Description: "Récupère les réponses à un message dans un canal",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id":    map[string]string{"type": "string", "description": "ID de l'équipe"},
					"channel_id": map[string]string{"type": "string", "description": "ID du canal"},
					"message_id": map[string]string{"type": "string", "description": "ID du message"},
				},
				"required": []string{"team_id", "channel_id", "message_id"},
			},
		},
		{
			Name:        "get_installed_apps",
			Description: "Liste les applications Teams installées par l'utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]string{"type": "string", "description": "ID de l'utilisateur"},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_chat_members",
			Description: "Liste les membres d'une conversation Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_id": map[string]string{"type": "string", "description": "ID de la conversation"},
				},
				"required": []string{"chat_id"},
			},
		},
	}
}
