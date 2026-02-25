package services

type MicrosoftTool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

func GetMicrosoftTools() []MicrosoftTool {
	return []MicrosoftTool{
		// === CALENDRIER ===
		{
			Name:        "get_calendar_events",
			Description: "Récupère les événements du calendrier d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_calendars",
			Description: "Récupère la liste des calendriers d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "create_meeting",
			Description: "Crée une réunion Teams dans le calendrier d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'organisateur",
					},
					"subject": map[string]interface{}{
						"type":        "string",
						"description": "Le sujet de la réunion",
					},
					"start_time": map[string]interface{}{
						"type":        "string",
						"description": "Heure de début (format ISO 8601: 2024-01-15T10:00:00)",
					},
					"end_time": map[string]interface{}{
						"type":        "string",
						"description": "Heure de fin (format ISO 8601: 2024-01-15T11:00:00)",
					},
					"attendees": map[string]interface{}{
						"type":        "array",
						"description": "Liste des emails des participants",
						"items":       map[string]interface{}{"type": "string"},
					},
				},
				"required": []string{"user_id", "subject", "start_time", "end_time"},
			},
		},
		{
			Name:        "find_meeting_times",
			Description: "Trouve des créneaux disponibles pour une réunion",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"attendees": map[string]interface{}{
						"type":        "array",
						"description": "Liste des emails des participants",
						"items":       map[string]interface{}{"type": "string"},
					},
					"duration_minutes": map[string]interface{}{
						"type":        "integer",
						"description": "Durée de la réunion en minutes",
					},
				},
				"required": []string{"attendees", "duration_minutes"},
			},
		},
		// === MESSAGERIE ===
		{
			Name:        "send_email",
			Description: "Envoie un email via Outlook",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"from": map[string]interface{}{
						"type":        "string",
						"description": "Email ou ID de l'expéditeur",
					},
					"to": map[string]interface{}{
						"type":        "string",
						"description": "Email du destinataire",
					},
					"subject": map[string]interface{}{
						"type":        "string",
						"description": "Sujet de l'email",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "Corps de l'email",
					},
				},
				"required": []string{"from", "to", "subject", "body"},
			},
		},
		{
			Name:        "get_important_emails",
			Description: "Récupère les emails importants d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_emails_from",
			Description: "Récupère les emails reçus d'un expéditeur spécifique",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
					"from_email": map[string]interface{}{
						"type":        "string",
						"description": "Email de l'expéditeur à filtrer",
					},
				},
				"required": []string{"user_id", "from_email"},
			},
		},
		{
			Name:        "forward_email",
			Description: "Transfère un email à un autre destinataire",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
					"message_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du message à transférer",
					},
					"to_email": map[string]interface{}{
						"type":        "string",
						"description": "Email du destinataire",
					},
					"comment": map[string]interface{}{
						"type":        "string",
						"description": "Commentaire à ajouter",
					},
				},
				"required": []string{"user_id", "message_id", "to_email"},
			},
		},
		{
			Name:        "get_email_delta",
			Description: "Récupère les nouveaux emails depuis la dernière synchronisation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"user_id"},
			},
		},
		// === UTILISATEURS ===
		{
			Name:        "get_users",
			Description: "Récupère la liste des utilisateurs de l'organisation",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_user_presence",
			Description: "Récupère la présence/disponibilité d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"user_id"},
			},
		},
		// === TEAMS ===
		{
			Name:        "get_teams",
			Description: "Récupère les équipes Teams d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
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
					"display_name": map[string]interface{}{
						"type":        "string",
						"description": "Nom de l'équipe",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description de l'équipe",
					},
					"visibility": map[string]interface{}{
						"type":        "string",
						"description": "Visibilité: 'public' ou 'private'",
					},
				},
				"required": []string{"display_name"},
			},
		},
		{
			Name:        "get_team_members",
			Description: "Récupère les membres d'une équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
				},
				"required": []string{"team_id"},
			},
		},
		{
			Name:        "get_team_channels",
			Description: "Récupère les canaux d'une équipe Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
				},
				"required": []string{"team_id"},
			},
		},
		{
			Name:        "get_channel_info",
			Description: "Récupère les informations d'un canal Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du canal",
					},
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
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
					"display_name": map[string]interface{}{
						"type":        "string",
						"description": "Nom du canal",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description du canal",
					},
					"membership_type": map[string]interface{}{
						"type":        "string",
						"description": "Type: 'standard' ou 'private'",
					},
				},
				"required": []string{"team_id", "display_name"},
			},
		},
		{
			Name:        "get_team_apps",
			Description: "Récupère les applications installées dans une équipe",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
				},
				"required": []string{"team_id"},
			},
		},
		{
			Name:        "create_chat",
			Description: "Crée un nouveau chat Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_type": map[string]interface{}{
						"type":        "string",
						"description": "Type: 'oneOnOne' ou 'group'",
					},
					"members": map[string]interface{}{
						"type":        "array",
						"description": "Liste des IDs Azure AD des membres",
						"items":       map[string]interface{}{"type": "string"},
					},
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "Sujet du chat (pour les groupes)",
					},
				},
				"required": []string{"chat_type", "members"},
			},
		},
		{
			Name:        "send_chat_message",
			Description: "Envoie un message dans un chat Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du chat",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Le message à envoyer",
					},
				},
				"required": []string{"chat_id", "message"},
			},
		},
		{
			Name:        "send_channel_message",
			Description: "Envoie un message dans un canal Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du canal",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Le message à envoyer",
					},
				},
				"required": []string{"team_id", "channel_id", "message"},
			},
		},
		// === GROUPES ===
		{
			Name:        "get_groups",
			Description: "Récupère tous les groupes Microsoft 365",
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
					"display_name": map[string]interface{}{
						"type":        "string",
						"description": "Nom du groupe",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Description du groupe",
					},
					"mail_nickname": map[string]interface{}{
						"type":        "string",
						"description": "Alias email du groupe",
					},
					"mail_enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Activer la messagerie",
					},
					"security_enabled": map[string]interface{}{
						"type":        "boolean",
						"description": "Activer la sécurité",
					},
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
					"group_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du groupe",
					},
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
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
					"group_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du groupe",
					},
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"group_id", "user_id"},
			},
		},
		{
			Name:        "delete_group",
			Description: "Supprime un groupe Microsoft 365",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"group_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du groupe à supprimer",
					},
				},
				"required": []string{"group_id"},
			},
		},
		{
			Name:        "get_my_groups",
			Description: "Récupère les groupes d'un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
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
					"group_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du groupe",
					},
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
					"group_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du groupe",
					},
				},
				"required": []string{"group_id"},
			},
		},
		// === TEAMS BETA ===
		{
			Name:        "get_channel_messages",
			Description: "Récupère les messages d'un canal Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du canal",
					},
				},
				"required": []string{"team_id", "channel_id"},
			},
		},
		{
			Name:        "get_message_replies",
			Description: "Récupère les réponses à un message de canal",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID de l'équipe",
					},
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du canal",
					},
					"message_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du message",
					},
				},
				"required": []string{"team_id", "channel_id", "message_id"},
			},
		},
		{
			Name:        "get_installed_apps",
			Description: "Récupère les applications Teams installées pour un utilisateur",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID Azure AD de l'utilisateur",
					},
				},
				"required": []string{"user_id"},
			},
		},
		{
			Name:        "get_chat_members",
			Description: "Récupère les membres d'un chat Teams",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chat_id": map[string]interface{}{
						"type":        "string",
						"description": "L'ID du chat",
					},
				},
				"required": []string{"chat_id"},
			},
		},
	}
}
