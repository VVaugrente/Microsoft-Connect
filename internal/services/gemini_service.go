package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"

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
	MaxOutputTokens    int                 `json:"maxOutputTokens,omitempty"`
	Temperature        float64             `json:"temperature,omitempty"`
	ThinkingConfig     *ThinkingConfig     `json:"thinkingConfig,omitempty"`
	ResponseModalities []string            `json:"responseModalities,omitempty"`
	SpeechConfig       *GeminiSpeechConfig `json:"speechConfig,omitempty"`
}

type ThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text             string              `json:"text,omitempty"`
	Thought          bool                `json:"thought,omitempty"`
	FunctionCall     *GeminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResp `json:"functionResponse,omitempty"`
	InlineData       *GeminiInlineData   `json:"inlineData,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GeminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type GeminiFunctionResp struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type GeminiSpeechConfig struct {
	VoiceConfig *GeminiVoiceConfig `json:"voiceConfig,omitempty"`
}

type GeminiVoiceConfig struct {
	PrebuiltVoiceConfig *GeminiPrebuiltVoice `json:"prebuiltVoiceConfig,omitempty"`
}

type GeminiPrebuiltVoice struct {
	VoiceName string `json:"voiceName"`
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
	FinishReason string        `json:"finishReason"`
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

// SendAudioMessage - Envoie de l'audio PCM (base64) à Gemini 2.5 et retourne la réponse audio PCM
func (s *GeminiService) SendAudioMessage(audioBase64 string, conversationID string, graphService *GraphService) ([]byte, error) {

	history := s.conversationStore.GetHistory(conversationID)
	contents := []GeminiContent{}

	// Reconstruire l'historique textuel
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

	// Ajouter l'audio courant comme message user
	contents = append(contents, GeminiContent{
		Role: "user",
		Parts: []GeminiPart{
			{
				InlineData: &GeminiInlineData{
					MimeType: "audio/pcm;rate=16000",
					Data:     audioBase64,
				},
			},
		},
	})

	reqBody := GeminiRequest{
		Contents: contents,
		SystemInstruction: &GeminiSystemInstruc{
			Parts: []GeminiPart{{Text: `Tu es NEO, un assistant vocal Microsoft 365.
Réponds de manière concise et claire en français.
Tu es en conversation vocale, évite les longues listes ou tableaux.`}},
		},
		Tools: []GeminiTool{{
			FunctionDeclarations: GetGeminiTools(),
		}},
		GenerationConfig: &GeminiGenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &GeminiSpeechConfig{
				VoiceConfig: &GeminiVoiceConfig{
					PrebuiltVoiceConfig: &GeminiPrebuiltVoice{
						VoiceName: "Aoede",
					},
				},
			},
			MaxOutputTokens: 1024,
			Temperature:     0.7,
			ThinkingConfig:  &ThinkingConfig{ThinkingBudget: 0},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audio request: %w", err)
	}

	apiURL := fmt.Sprintf("%s?key=%s", geminiBaseURL, s.apiKey)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audio request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Gemini audio error %d: %s", resp.StatusCode, string(body))
	}

	var result GeminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse audio response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("Gemini error %d: %s", result.Error.Code, result.Error.Message)
	}

	if len(result.Candidates) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	// Extraire l'audio de la réponse
	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != "" {
			audioBytes, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to decode audio response: %w", err)
			}
			log.Printf("[GeminiAudio] Réponse audio: %d bytes pour conversationID: %s", len(audioBytes), conversationID)

			// Sauvegarder dans l'historique
			s.conversationStore.AddMessage(conversationID, "assistant", "[audio_response]")
			return audioBytes, nil
		}
	}

	// Gemini a répondu en texte au lieu d'audio → log et retourner nil
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			log.Printf("[GeminiAudio] Réponse texte inattendue: %s", part.Text)
			s.conversationStore.AddMessage(conversationID, "assistant", part.Text)
		}
	}

	return nil, nil
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
				ThinkingConfig: &ThinkingConfig{
					ThinkingBudget: 0,
				},
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

		// Séparer les parts: thoughts, function calls, texte
		var textParts []string
		var functionCalls []*GeminiFunctionCall

		for _, part := range candidate.Content.Parts {
			if part.Thought {
				continue // Ignorer les pensées internes
			}
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}

		// Si on a des function calls, les exécuter
		if len(functionCalls) > 0 {
			contents = append(contents, candidate.Content)

			funcResponses := []GeminiPart{}
			for _, fc := range functionCalls {
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

			contents = append(contents, GeminiContent{
				Role:  "user",
				Parts: funcResponses,
			})
			continue
		}

		// Si on a du texte, le retourner
		if len(textParts) > 0 {
			return strings.Join(textParts, "\n"), nil
		}

		// STOP sans texte ni function call
		switch candidate.FinishReason {
		case "STOP":
			return "", fmt.Errorf("Gemini s'est arrêté sans réponse")
		case "MAX_TOKENS":
			return strings.Join(textParts, "\n"), nil
		case "SAFETY":
			return "⚠️ Je ne peux pas répondre à cette demande.", nil
		case "MALFORMED_FUNCTION_CALL":
			return "", fmt.Errorf("erreur appel de fonction Gemini")
		default:
			return "", fmt.Errorf("unexpected finish reason: %s", candidate.FinishReason)
		}
	}
}
