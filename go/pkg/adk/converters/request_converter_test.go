package converters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

func TestRequestConverter_Convert(t *testing.T) {
	converter := NewRequestConverter()

	tests := []struct {
		name       string
		requestCtx *RequestContext
		wantErr    bool
		checkFunc  func(t *testing.T, args *RunArgs)
	}{
		{
			name: "basic conversion",
			requestCtx: &RequestContext{
				SessionID: "session-123",
				UserID:    "user-456",
				TaskID:    "task-789",
				ContextID: "ctx-abc",
				Message: &protocol.Message{
					MessageID: "msg-1",
					Kind:      protocol.KindMessage,
					Parts: []protocol.Part{
						&protocol.TextPart{Text: "Hello"},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, args *RunArgs) {
				assert.Equal(t, "user-456", args.UserID)
				assert.Equal(t, "session-123", args.SessionID)
				require.NotNil(t, args.NewMessage)
				assert.Len(t, args.NewMessage.Parts, 1)
			},
		},
		{
			name: "fallback to context ID for user",
			requestCtx: &RequestContext{
				SessionID: "session-123",
				TaskID:    "task-789",
				ContextID: "ctx-abc",
				Message: &protocol.Message{
					Parts: []protocol.Part{
						&protocol.TextPart{Text: "Test"},
					},
				},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, args *RunArgs) {
				assert.Equal(t, "ctx-abc", args.UserID) // Falls back to ContextID
				assert.Equal(t, "session-123", args.SessionID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := converter.Convert(tt.requestCtx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, args)
			if tt.checkFunc != nil {
				tt.checkFunc(t, args)
			}
		})
	}
}
