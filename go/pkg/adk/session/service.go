package session

import (
	"context"
)

// Service defines the interface for session management
type Service interface {
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*Session, error)
	GetSession(ctx context.Context, appName, userID, sessionID string) (*Session, error)
	ListSessions(ctx context.Context, appName, userID string) ([]*Session, error)
	AppendEvent(ctx context.Context, session *Session, event *Event) error
	DeleteSession(ctx context.Context, appName, userID, sessionID string) error
}
