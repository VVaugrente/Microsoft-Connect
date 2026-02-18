package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
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

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
