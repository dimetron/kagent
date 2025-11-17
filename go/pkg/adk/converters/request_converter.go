package converters

import (
	"fmt"

	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// RequestConverter handles conversion from A2A requests to ADK RunArgs
type RequestConverter struct {
	partConverter *PartConverter
}

// NewRequestConverter creates a new RequestConverter
func NewRequestConverter() *RequestConverter {
	return &RequestConverter{
		partConverter: NewPartConverter(),
	}
}

// Convert converts an A2A request context to ADK RunArgs
func (c *RequestConverter) Convert(ctx *RequestContext) (*RunArgs, error) {
	// Extract user ID (from auth context or generate from context_id)
	userID := ctx.UserID
	if userID == "" {
		userID = ctx.ContextID
	}

	// Extract session ID
	sessionID := ctx.SessionID
	if sessionID == "" {
		sessionID = ctx.TaskID
	}

	// Convert message to Content
	var content *Content
	var err error

	// Handle different message types
	switch msg := ctx.Message.(type) {
	case *protocol.Message:
		content, err = c.convertA2AMessage(msg)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrCodeConversion, "failed to convert message", err)
		}

	case protocol.Message:
		content, err = c.convertA2AMessage(&msg)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrCodeConversion, "failed to convert message", err)
		}

	default:
		return nil, apperrors.New(apperrors.ErrCodeConversion,
			fmt.Sprintf("unsupported message type: %T", ctx.Message), nil)
	}

	return &RunArgs{
		UserID:     userID,
		SessionID:  sessionID,
		NewMessage: content,
		RunConfig:  &RunConfig{},
	}, nil
}

func (c *RequestConverter) convertA2AMessage(msg *protocol.Message) (*Content, error) {
	// Convert parts
	parts, err := c.partConverter.ConvertA2AToContent(msg.Parts)
	if err != nil {
		return nil, err
	}

	// Determine role from message metadata or default to user
	role := "user"
	if msg.Metadata != nil {
		if r, ok := msg.Metadata["role"].(string); ok {
			role = r
		}
	}

	return &Content{
		Role:  role,
		Parts: parts,
	}, nil
}

// ConvertMessageHistory converts a history of A2A messages to Content
func (c *RequestConverter) ConvertMessageHistory(messages []protocol.Message) ([]*Content, error) {
	var contents []*Content

	for _, msg := range messages {
		content, err := c.convertA2AMessage(&msg)
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}

	return contents, nil
}
