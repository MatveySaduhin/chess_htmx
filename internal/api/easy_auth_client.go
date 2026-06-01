package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EasyAuthClient struct {
	baseURL    string
	httpClient *http.Client
}

type EasyAuthRegisterResponse struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

type EasyAuthLoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	SessionID    string `json:"session_id"`
}

type EasyAuthError struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

func (e EasyAuthError) Error() string {
	return fmt.Sprintf("easy auth returned %d: %s", e.StatusCode, e.Body)
}

func NewEasyAuthClient(config EasyAuthConfig) *EasyAuthClient {
	return &EasyAuthClient{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *EasyAuthClient) Register(ctx context.Context, email, password string) (EasyAuthRegisterResponse, error) {
	var response EasyAuthRegisterResponse
	err := c.postJSON(ctx, "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	}, http.StatusCreated, &response)
	return response, err
}

func (c *EasyAuthClient) Login(ctx context.Context, email, password string) (EasyAuthLoginResponse, error) {
	var response EasyAuthLoginResponse
	err := c.postJSON(ctx, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, http.StatusOK, &response)
	return response, err
}

func (c *EasyAuthClient) postJSON(ctx context.Context, path string, payload any, expectedStatus int, output any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode != expectedStatus {
		return EasyAuthError{
			StatusCode: resp.StatusCode,
			Body:       string(responseBody),
		}
	}

	if err := json.Unmarshal(responseBody, output); err != nil {
		return fmt.Errorf("decode easy auth response: %w", err)
	}

	return nil
}
