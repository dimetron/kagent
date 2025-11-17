package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
)

// KAgentService implements the Service interface using KAgent HTTP API
type KAgentService struct {
	baseURL    string
	httpClient *http.Client
	tokenFunc  func() string // Function to get current token
}

// NewKAgentService creates a new KAgentService
func NewKAgentService(baseURL string, tokenFunc func() string) *KAgentService {
	return &KAgentService{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		tokenFunc:  tokenFunc,
	}
}

func (k *KAgentService) addAuthHeaders(req *http.Request) {
	if k.tokenFunc != nil {
		if token := k.tokenFunc(); token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
	}
}

func (k *KAgentService) CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionCreate, "failed to marshal request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", k.baseURL+"/api/sessions", bytes.NewReader(data))
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionCreate, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	k.addAuthHeaders(httpReq)

	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionCreate, "failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, apperrors.New(apperrors.ErrCodeSessionCreate,
			fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)), nil)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionCreate, "failed to decode response", err)
	}

	return &session, nil
}

func (k *KAgentService) GetSession(ctx context.Context, appName, userID, sessionID string) (*Session, error) {
	url := fmt.Sprintf("%s/api/sessions/%s?app_name=%s&user_id=%s",
		k.baseURL, sessionID, url.QueryEscape(appName), url.QueryEscape(userID))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionGet, "failed to create request", err)
	}

	k.addAuthHeaders(httpReq)

	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionGet, "failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, apperrors.New(apperrors.ErrCodeSessionGet,
			fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)), nil)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionGet, "failed to decode response", err)
	}

	return &session, nil
}

func (k *KAgentService) ListSessions(ctx context.Context, appName, userID string) ([]*Session, error) {
	url := fmt.Sprintf("%s/api/sessions?app_name=%s&user_id=%s",
		k.baseURL, url.QueryEscape(appName), url.QueryEscape(userID))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionGet, "failed to create request", err)
	}

	k.addAuthHeaders(httpReq)

	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionGet, "failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, apperrors.New(apperrors.ErrCodeSessionGet,
			fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)), nil)
	}

	var sessions []*Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, apperrors.New(apperrors.ErrCodeSessionGet, "failed to decode response", err)
	}

	return sessions, nil
}

func (k *KAgentService) AppendEvent(ctx context.Context, session *Session, event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeSessionGet, "failed to marshal event", err)
	}

	url := fmt.Sprintf("%s/api/sessions/%s/events", k.baseURL, session.ID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return apperrors.New(apperrors.ErrCodeSessionGet, "failed to create request", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	k.addAuthHeaders(httpReq)

	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeSessionGet, "failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return apperrors.New(apperrors.ErrCodeSessionGet,
			fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)), nil)
	}

	return nil
}

func (k *KAgentService) DeleteSession(ctx context.Context, appName, userID, sessionID string) error {
	url := fmt.Sprintf("%s/api/sessions/%s?app_name=%s&user_id=%s",
		k.baseURL, sessionID, url.QueryEscape(appName), url.QueryEscape(userID))

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeSessionDelete, "failed to create request", err)
	}

	k.addAuthHeaders(httpReq)

	resp, err := k.httpClient.Do(httpReq)
	if err != nil {
		return apperrors.New(apperrors.ErrCodeSessionDelete, "failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return apperrors.New(apperrors.ErrCodeSessionDelete,
			fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body)), nil)
	}

	return nil
}
