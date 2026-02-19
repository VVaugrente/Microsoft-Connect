package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"microsoft_connector/config"
)

type AuthService struct {
	config      *config.Config
	accessToken string
	expiresAt   time.Time
	mu          sync.RWMutex
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{
		config: cfg,
	}
}

func (s *AuthService) GetAccessToken() (string, error) {
	s.mu.RLock()
	if s.accessToken != "" && time.Now().Before(s.expiresAt.Add(-5*time.Minute)) {
		token := s.accessToken
		s.mu.RUnlock()
		return token, nil
	}
	s.mu.RUnlock()

	return s.refreshToken()
}

func (s *AuthService) refreshToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check après avoir obtenu le lock
	if s.accessToken != "" && time.Now().Before(s.expiresAt.Add(-5*time.Minute)) {
		return s.accessToken, nil
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", s.config.TenantID)

	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default") // <-- IMPORTANT: Graph scope
	data.Set("grant_type", "client_credentials")

	log.Printf("=== GRAPH TOKEN REQUEST ===")
	log.Printf("URL: %s", tokenURL)
	log.Printf("ClientID: %s", s.config.ClientID)
	log.Printf("Scope: https://graph.microsoft.com/.default")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("=== GRAPH TOKEN RESPONSE ===")
	log.Printf("Status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	log.Printf("Graph token obtained successfully, expires in %d seconds", tokenResp.ExpiresIn)

	s.accessToken = tokenResp.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return s.accessToken, nil
}
