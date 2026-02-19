package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	graphBaseURL     = "https://graph.microsoft.com/v1.0"
	graphBetaBaseURL = "https://graph.microsoft.com/beta"
)

type GraphService struct {
	authService *AuthService
	httpClient  *http.Client
}

func NewGraphService(authService *AuthService) *GraphService {
	return &GraphService{
		authService: authService,
		httpClient:  &http.Client{},
	}
}

func (s *GraphService) Get(endpoint string) (map[string]any, error) {
	return s.request("GET", graphBaseURL+endpoint, nil)
}

func (s *GraphService) GetBeta(endpoint string) (map[string]any, error) {
	return s.request("GET", graphBetaBaseURL+endpoint, nil)
}

func (s *GraphService) Post(endpoint string, body map[string]any) (map[string]any, error) {
	return s.request("POST", graphBaseURL+endpoint, body)
}

func (s *GraphService) PostBeta(endpoint string, body map[string]any) (map[string]any, error) {
	return s.request("POST", graphBetaBaseURL+endpoint, body)
}

func (s *GraphService) Delete(endpoint string) error {
	_, err := s.request("DELETE", graphBaseURL+endpoint, nil)
	return err
}

func (s *GraphService) request(method, url string, body map[string]any) (map[string]any, error) {
	token, err := s.authService.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	log.Printf("=== GRAPH API REQUEST ===")
	log.Printf("Method: %s", method)
	log.Printf("URL: %s", url)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
		log.Printf("Request body: %s", string(jsonBody))
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ConsistencyLevel", "eventual")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("=== GRAPH API RESPONSE ===")
	log.Printf("Status: %d", resp.StatusCode)

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("Response body: %s", string(bodyBytes))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Gérer les réponses sans contenu (204 No Content et 202 Accepted)
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}

	// Si le body est vide, retourner nil sans erreur
	if len(bodyBytes) == 0 {
		return nil, nil
	}

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
