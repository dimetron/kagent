package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	apperrors "github.com/kagent-dev/kagent/go/pkg/app/errors"
)

const (
	DefaultTokenPath     = "/var/run/secrets/tokens/kagent-token"
	DefaultRefreshPeriod = 60 * time.Second
)

// TokenService manages authentication tokens
type TokenService struct {
	appName       string
	tokenPath     string
	refreshPeriod time.Duration
	token         string
	mu            sync.RWMutex
	stopCh        chan struct{}
}

// NewTokenService creates a new TokenService
func NewTokenService(appName, tokenPath string) *TokenService {
	if tokenPath == "" {
		tokenPath = DefaultTokenPath
	}
	return &TokenService{
		appName:       appName,
		tokenPath:     tokenPath,
		refreshPeriod: DefaultRefreshPeriod,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the token refresh cycle
func (t *TokenService) Start(ctx context.Context) error {
	// Load token initially
	if err := t.refreshToken(); err != nil {
		return apperrors.New(apperrors.ErrCodeAuthFailed, "failed to load initial token", err)
	}

	// Start refresh ticker
	ticker := time.NewTicker(t.refreshPeriod)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := t.refreshToken(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to refresh token: %v\n", err)
				}
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-t.stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

// Stop stops the token refresh cycle
func (t *TokenService) Stop() {
	close(t.stopCh)
}

func (t *TokenService) refreshToken() error {
	data, err := os.ReadFile(t.tokenPath)
	if err != nil {
		// If token file doesn't exist, it's not necessarily an error
		// (e.g., in local development)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	t.mu.Lock()
	t.token = string(data)
	t.mu.Unlock()

	return nil
}

// GetToken returns the current token
func (t *TokenService) GetToken() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.token
}

// AddHeaders adds authentication headers to an HTTP request
func (t *TokenService) AddHeaders(req *http.Request) {
	token := t.GetToken()
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	if t.appName != "" {
		req.Header.Set("X-Agent-Name", t.appName)
	}
}
