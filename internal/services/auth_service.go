package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
	return &AuthService{config: cfg}
}

func (s *AuthService) GetAccessToken() (string, error) {
	s.mu.RLock()
	if s.accessToken != "" && time.Now().Before(s.expiresAt) {
		defer s.mu.RUnlock()
		return s.accessToken, nil
	}
	s.mu.RUnlock()

	return s.refreshToken()
}

func (s *AuthService) refreshToken() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", s.config.TenantID)

	data := url.Values{}
	data.Set("client_id", s.config.ClientID)
	data.Set("client_secret", s.config.ClientSecret)
	data.Set("scope", "https://graph.microsoft.com/.default")
	data.Set("grant_type", "client_credentials")

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned status %d", resp.StatusCode)
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	s.accessToken = tokenResp.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return s.accessToken, nil
}
