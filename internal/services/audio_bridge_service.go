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

type AudioBridgeService struct {
	baseURL    string
	httpClient *http.Client
}

type JoinCallRequest struct {
	JoinUrl     string `json:"joinUrl"`
	DisplayName string `json:"displayName"`
}

type JoinCallResponse struct {
	CallID     string `json:"callId"`
	ThreadID   string `json:"threadId"`
	ScenarioID string `json:"scenarioId"`
	Port       string `json:"port"`
}

type ActiveCall struct {
	ThreadID     string `json:"threadId"`
	CallID       string `json:"callId"`
	Participants int    `json:"participants"`
}

func NewAudioBridgeService(baseURL string) *AudioBridgeService {
	return &AudioBridgeService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// JoinCall - Demande au C# de rejoindre un appel vocal Teams
func (s *AudioBridgeService) JoinCall(joinURL, displayName string) (*JoinCallResponse, error) {
	log.Printf("[AudioBridge] Demande de rejoindre l'appel: %s", joinURL)

	payload := JoinCallRequest{
		JoinUrl:     joinURL,
		DisplayName: displayName,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.httpClient.Post(
		s.baseURL+"/calls",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call C# bridge: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("C# bridge error %d: %s", resp.StatusCode, string(respBody))
	}

	var joinResp JoinCallResponse
	if err := json.Unmarshal(respBody, &joinResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("[AudioBridge] Appel rejoint. CallID: %s, ThreadID: %s", joinResp.CallID, joinResp.ThreadID)
	return &joinResp, nil
}

// LeaveCall - Demande au C# de quitter un appel
func (s *AudioBridgeService) LeaveCall(threadID string) error {
	log.Printf("[AudioBridge] Demande de quitter l'appel: %s", threadID)

	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/calls?threadId=%s", s.baseURL, threadID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call C# bridge: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("C# bridge error %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[AudioBridge] Appel quitté: %s", threadID)
	return nil
}

// GetActiveCalls - Récupère les appels actifs
func (s *AudioBridgeService) GetActiveCalls() ([]ActiveCall, error) {
	resp, err := s.httpClient.Get(s.baseURL + "/calls")
	if err != nil {
		return nil, fmt.Errorf("failed to get active calls: %w", err)
	}
	defer resp.Body.Close()

	var calls []ActiveCall
	if err := json.NewDecoder(resp.Body).Decode(&calls); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return calls, nil
}

// IsHealthy - Vérifie que le C# est joignable
func (s *AudioBridgeService) IsHealthy() bool {
	resp, err := s.httpClient.Get(s.baseURL + "/health")
	if err != nil {
		log.Printf("[AudioBridge] Health check failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
