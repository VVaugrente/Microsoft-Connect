package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-native-audio-preview-12-2025"

type GeminiService struct {
	apiKey            string
	httpClient        *http.Client
	conversationStore *ConversationStore
}

// ===== Structures Request =====

type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"`
	Tools             []GeminiTool            `json:"tools,omitempty"`
	SystemInstruction *GeminiSystemInstruc    `json:"system_instruction,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiSystemInstruc struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" ou "model"
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text             string              `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResp `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type GeminiFunctionResp struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

// ===== Structures Response =====

type GeminiResponse struct {
	Candidates     []GeminiCandidate `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason,omitempty"`
	} `json:"promptFeedback,omitempty"`
	Error *GeminiError `json:"error,omitempty"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"` // "STOP", "MAX_TOKENS", "SAFETY"
	Index        int           `json:"index"`
}

type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// ===== Structures Tools =====

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDecl `json:"function_declarations"`
}

type GeminiFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ===== Constructor =====

func NewGeminiService(apiKey string) *GeminiService {
	return &GeminiService{
		apiKey:            apiKey,
		httpClient:        &http.Client{},
		conversationStore: NewConversationStore(),
	}
}

// ===== Public Methods =====

func (s *GeminiService) SendMessageWithContext(userMessage string, context string, conversationID string, graphService *GraphService) (string, error) {
	history := s.conversationStore.GetHistory(conversationID)

	contents := []GeminiContent{}

	// Reconstruire l'historique (role: "user" ou "model")
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: msg.Content}},
		})
	}

	// Ajouter le nouveau message utilisateur
	contents = append(contents, GeminiContent{
		Role:  "user",
		Parts: []GeminiPart{{Text: userMessage}},
	})

	// Sauvegarder avant l'envoi
	s.conversationStore.AddMessage(conversationID, "user", userMessage)

	response, err := s.sendWithTools(contents, context, graphService)
	if err != nil {
		return "", err
	}

	s.conversationStore.AddMessage(conversationID, "assistant", response)
	return response, nil
}

// ===== Private Methods =====

func (s *GeminiService) sendWithTools(contents []GeminiContent, systemContext string, graphService *GraphService) (string, error) {
	executor := &ToolExecutor{}

	for {
		reqBody := GeminiRequest{
			Contents: contents,
			SystemInstruction: &GeminiSystemInstruc{
				Parts: []GeminiPart{{Text: systemContext}},
			},
			Tools: []GeminiTool{{
				FunctionDeclarations: GetGeminiTools(),
			}},
			GenerationConfig: &GeminiGenerationConfig{
				MaxOutputTokens: 4096,
				Temperature:     0.7,
			},
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		apiURL := fmt.Sprintf("%s?key=%s", geminiBaseURL, s.apiKey)
		req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("request failed: %w", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, string(body))
		}

		var result GeminiResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("failed to parse response: %w", err)
		}

		// Vérifier les erreurs API
		if result.Error != nil {
			return "", fmt.Errorf("Gemini error %d: %s", result.Error.Code, result.Error.Message)
		}

		if len(result.Candidates) == 0 {
			return "", fmt.Errorf("empty response from Gemini")
		}

		candidate := result.Candidates[0]

		switch candidate.FinishReason {
		case "STOP":
			// Réponse texte finale
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					return part.Text, nil
				}
			}
			return "", fmt.Errorf("no text in STOP response")

		case "":
			// Gemini retourne "" pour les function calls
			fallthrough
		case "FUNCTION_CALL":
			// Ajouter la réponse du modèle à l'historique de la conversation
			contents = append(contents, candidate.Content)

			// Exécuter tous les function calls en parallèle
			funcResponses := []GeminiPart{}
			for _, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					fc := part.FunctionCall
					log.Printf("=== GEMINI FUNCTION CALL: %s ===", fc.Name)

					inputJSON, _ := json.Marshal(fc.Args)
					toolResult := executor.Execute(fc.Name, inputJSON, graphService)

					log.Printf("=== TOOL RESULT: %s ===", toolResult)

					funcResponses = append(funcResponses, GeminiPart{
						FunctionResponse: &GeminiFunctionResp{
							Name:     fc.Name,
							Response: map[string]any{"result": toolResult},
						},
					})
				}
			}

			if len(funcResponses) == 0 {
				return "", fmt.Errorf("no function call found in response")
			}

			// Ajouter les résultats des tools comme message "user"
			contents = append(contents, GeminiContent{
				Role:  "user",
				Parts: funcResponses,
			})

		case "MAX_TOKENS":
			return "", fmt.Errorf("response truncated: max tokens reached")

		case "SAFETY":
			return "Désolé, je ne peux pas répondre à cette demande.", nil

		default:
			return "", fmt.Errorf("unexpected finish reason: %s", candidate.FinishReason)
		}
	}
}
