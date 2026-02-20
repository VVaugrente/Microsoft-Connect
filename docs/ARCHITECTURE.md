# Architecture Microsoft Connector

## 1. Vue d'ensemble de l'application

```mermaid
flowchart TB
    subgraph CLIENTS["🖥️ Clients"]
        TEAMS[Microsoft Teams]
        API_CLIENT[API REST Client]
    end

    subgraph CONNECTOR["🔌 Microsoft Connector"]
        GIN[Gin Router]
        
        subgraph HANDLERS["📦 Handlers"]
            BOT[BotHandler]
            TEAMS_BOT[TeamsBotHandler]
            CHAT[ChatHandler]
            USERS[UsersHandler]
            TEAMS_H[TeamsHandler]
            CAL[CalendarHandler]
            MAIL[MailHandler]
            GROUPS[GroupsHandler]
            BATCH[BatchHandler]
        end
        
        subgraph SERVICES["⚙️ Services"]
            AUTH[AuthService]
            GRAPH[GraphService]
            CLAUDE[ClaudeService]
            CONV[ConversationStore]
            TEAMS_CHAT[TeamsChatService]
        end
    end

    subgraph EXTERNAL["☁️ APIs Externes"]
        GRAPH_API[Microsoft Graph API]
        CLAUDE_API[Anthropic Claude API]
        BOT_FW[Bot Framework]
    end

    TEAMS -->|Bot Framework| BOT_FW
    BOT_FW -->|POST /api/messages| GIN
    API_CLIENT -->|REST| GIN
    
    GIN --> HANDLERS
    HANDLERS --> SERVICES
    
    AUTH -->|OAuth2 Token| GRAPH
    GRAPH -->|HTTP| GRAPH_API
    CLAUDE -->|HTTP| CLAUDE_API
    CLAUDE -->|Tool Calls| GRAPH
```

## 2. Flux d'authentification Microsoft

```mermaid
sequenceDiagram
    participant H as Handler
    participant GS as GraphService
    participant AS as AuthService
    participant AAD as Azure AD

    H->>GS: Get("/users")
    GS->>AS: GetAccessToken()
    
    alt Token en cache valide
        AS-->>GS: Token existant
    else Token expiré ou absent
        AS->>AAD: POST /oauth2/v2.0/token
        Note over AS,AAD: Client Credentials Flow<br/>client_id + client_secret
        AAD-->>AS: access_token + expires_in
        AS->>AS: Cache token
        AS-->>GS: Nouveau token
    end
    
    GS->>GS: Ajouter Header Authorization
    GS-->>H: Réponse Graph API
```

## 3. Flux du Bot Teams

```mermaid
sequenceDiagram
    participant U as Utilisateur Teams
    participant T as Microsoft Teams
    participant BF as Bot Framework
    participant B as BotHandler
    participant C as ClaudeService
    participant G as GraphService

    U->>T: @NEO quelle est ma prochaine réunion ?
    T->>BF: Activity (message)
    BF->>B: POST /api/messages
    
    B->>B: Valider JWT Token
    B->>B: Extraire message et contexte
    
    B->>C: Chat(conversationId, message)
    
    loop Boucle d'outils
        C->>C: Appel Claude API
        
        alt Claude demande un outil
            C->>G: Exécuter tool (ex: get_calendar_events)
            G-->>C: Résultat de l'outil
            C->>C: Renvoyer résultat à Claude
        else Réponse finale
            C-->>B: Texte de réponse
        end
    end
    
    B->>BF: Reply Activity
    BF->>T: Réponse
    T->>U: Affichage message
```

## 4. Architecture des Services

```mermaid
classDiagram
    class AuthService {
        -config Config
        -accessToken string
        -tokenExpiry time.Time
        -mutex sync.RWMutex
        +GetAccessToken() string, error
        -refreshToken() error
    }

    class GraphService {
        -authService AuthService
        -httpClient http.Client
        +Get(endpoint) map, error
        +GetBeta(endpoint) map, error
        +Post(endpoint, body) map, error
        +Delete(endpoint) error
        +Patch(endpoint, body) map, error
    }

    class ClaudeService {
        -apiKey string
        -httpClient http.Client
        -conversationStore ConversationStore
        -graphService GraphService
        +Chat(convId, message, userId) string, error
        -executeTools(toolCalls) []ToolResult
        -callClaudeAPI(messages, tools) Response
    }

    class ConversationStore {
        -conversations map
        -mutex sync.RWMutex
        +AddMessage(convId, message)
        +GetHistory(convId) []Message
        +Clear(convId)
        -cleanup()
    }

    class TeamsChatService {
        -graphService GraphService
        +CreateOneOnOneChat(userId) Chat, error
        +SendMessage(chatId, content) error
        +GetOrCreateChat(userId) Chat, error
    }

    AuthService <-- GraphService : utilise
    GraphService <-- ClaudeService : utilise
    ConversationStore <-- ClaudeService : utilise
    GraphService <-- TeamsChatService : utilise
```

## 5. Routes API

```mermaid
flowchart LR
    subgraph ROUTES["🛣️ Routes API"]
        ROOT["/"]
        HEALTH["/health"]
        
        subgraph API["/api"]
            MESSAGES["/messages"]
            WEBHOOK["/webhook"]
            
            subgraph USERS_R["/users"]
                U1["GET /"]
                U2["GET /guests"]
                U3["GET /:email"]
            end
            
            subgraph TEAMS_R["/teams"]
                T1["POST /"]
                T2["GET /joined"]
                T3["GET /:id/members"]
                T4["GET /:id/channels"]
                T5["POST /:id/channels"]
                T6["POST /:id/channels/:channelId/messages"]
            end
            
            subgraph CALENDAR_R["/calendar"]
                C1["GET /week"]
                C2["GET /events"]
                C3["POST /events"]
                C4["POST /findMeetingTimes"]
            end
            
            subgraph MAIL_R["/mail"]
                M1["GET /important"]
                M2["GET /from/:email"]
                M3["POST /send"]
            end
            
            subgraph CHAT_R["/chat"]
                CH1["POST /"]
                CH2["POST /direct"]
                CH3["POST /channel"]
            end
            
            BATCH_R["/batch"]
        end
    end
```

## 6. Outils Claude disponibles

```mermaid
mindmap
  root((Claude Tools))
    Calendrier
      get_calendar_events
        Récupère les événements
        Paramètres: days
      create_meeting
        Crée une réunion Teams
        Paramètres: subject, attendees, start, end
    Communication
      send_email
        Envoie un email
        Paramètres: to, subject, body
      send_channel_message
        Message dans un canal Teams
        Paramètres: teamId, channelId, message
    Utilisateurs
      get_users
        Liste les utilisateurs
        Paramètres: filter
      get_user_presence
        Statut de présence
        Paramètres: userId
    Teams
      get_teams
        Liste les équipes
        Aucun paramètre
```

## 7. Gestion des conversations

```mermaid
stateDiagram-v2
    [*] --> NouvMessage: Nouveau message
    
    NouvMessage --> RecupHist: GetHistory(convId)
    
    RecupHist --> AjoutMsg: AddMessage(role, content)
    
    AjoutMsg --> AppelClaude: Messages + Tools
    
    AppelClaude --> ToolCall: tool_use
    AppelClaude --> Reponse: text
    
    ToolCall --> ExecuteTool: graphService call
    ExecuteTool --> AjoutResult: AddMessage(tool_result)
    AjoutResult --> AppelClaude
    
    Reponse --> AjoutReponse: AddMessage(assistant)
    AjoutReponse --> [*]
    
    note right of RecupHist
        Max 50 messages
        TTL 30 minutes
    end note
```

## 8. Flux de déploiement Render

```mermaid
flowchart TB
    subgraph RENDER["☁️ Render.com"]
        ENV[Variables d'environnement]
        SERVER[Serveur Go]
        
        subgraph SELF_PING["🔄 Self-Ping"]
            TICKER[Ticker 1 minute]
            PING[GET /health]
        end
    end
    
    subgraph STARTUP["🚀 Démarrage"]
        S1[Load Config]
        S2[Init Services]
        S3[Init Handlers]
        S4[Setup Routes]
        S5[Start Server]
        S6{RENDER_EXTERNAL_URL?}
        S7[Start Self-Ping]
    end
    
    S1 --> S2 --> S3 --> S4 --> S5 --> S6
    S6 -->|Oui| S7
    S6 -->|Non| FIN[Serveur prêt]
    S7 --> FIN
    
    TICKER --> PING
    PING --> SERVER
    
    note1[Évite le cold start<br/>sur plan gratuit Render]
```

## 9. Flux complet d'une requête

```mermaid
flowchart TB
    START((Requête HTTP))
    
    START --> GIN[Gin Router]
    GIN --> MATCH{Route match?}
    
    MATCH -->|Non| 404[404 Not Found]
    MATCH -->|Oui| HANDLER[Handler approprié]
    
    HANDLER --> PARSE[Parse Request]
    PARSE --> VALIDATE{Validation OK?}
    
    VALIDATE -->|Non| 400[400 Bad Request]
    VALIDATE -->|Oui| SERVICE[Appel Service]
    
    SERVICE --> AUTH{Besoin Auth?}
    AUTH -->|Oui| TOKEN[GetAccessToken]
    AUTH -->|Non| EXECUTE
    
    TOKEN --> VALID{Token valide?}
    VALID -->|Non| REFRESH[Refresh Token]
    REFRESH --> EXECUTE
    VALID -->|Oui| EXECUTE
    
    EXECUTE[Exécuter logique]
    EXECUTE --> API{API externe?}
    
    API -->|Graph| GRAPH_CALL[HTTP vers Graph API]
    API -->|Claude| CLAUDE_CALL[HTTP vers Claude API]
    API -->|Non| RESULT
    
    GRAPH_CALL --> RESULT[Résultat]
    CLAUDE_CALL --> RESULT
    
    RESULT --> FORMAT[Formater réponse JSON]
    FORMAT --> RESPONSE((Réponse HTTP))
```

## 10. Structure des fichiers

```mermaid
flowchart TB
    subgraph PROJECT["📁 microsoft_connector"]
        subgraph CMD["cmd/"]
            MAIN["server/main.go<br/>🚀 Point d'entrée"]
        end
        
        subgraph CONFIG["config/"]
            CFG["config.go<br/>⚙️ Configuration"]
        end
        
        subgraph INTERNAL["internal/"]
            subgraph HANDLERS["handlers/"]
                H1["bot_handler.go"]
                H2["teams_bot.go"]
                H3["chat.go"]
                H4["users.go"]
                H5["teams.go"]
                H6["calendar.go"]
                H7["mail.go"]
                H8["groups.go"]
                H9["batch.go"]
                H10["teams_beta.go"]
            end
            
            subgraph SERVICES["services/"]
                S1["auth_service.go"]
                S2["graph_service.go"]
                S3["claude_service.go"]
                S4["claude_tools.go"]
                S5["conversation_store.go"]
                S6["teams_chat_service.go"]
            end
        end
        
        MANIFEST["teams-manifest/<br/>📋 Manifest Bot Teams"]
    end
    
    MAIN --> CFG
    MAIN --> HANDLERS
    HANDLERS --> SERVICES
```

---

## Résumé

| Composant | Rôle |
|-----------|------|
| **main.go** | Initialise tout et démarre le serveur |
| **AuthService** | Gère les tokens OAuth2 Microsoft |
| **GraphService** | Client HTTP pour Microsoft Graph |
| **ClaudeService** | Intégration IA avec boucle d'outils |
| **ConversationStore** | Mémoire des conversations |
| **Handlers** | Contrôleurs REST pour chaque domaine |
| **Bot Framework** | Communication avec Teams |