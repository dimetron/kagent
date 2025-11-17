package converters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

func TestPartConverter_ConvertA2AToContent(t *testing.T) {
	converter := NewPartConverter()

	tests := []struct {
		name     string
		a2aParts []protocol.Part
		want     int
		wantErr  bool
	}{
		{
			name: "single text part",
			a2aParts: []protocol.Part{
				&protocol.TextPart{Text: "Hello world"},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "multiple parts",
			a2aParts: []protocol.Part{
				&protocol.TextPart{Text: "Part 1"},
				&protocol.TextPart{Text: "Part 2"},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "file part with URI",
			a2aParts: []protocol.Part{
				&protocol.FilePart{
					FileURI:  "file:///path/to/file.txt",
					MimeType: "text/plain",
				},
			},
			want:    1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := converter.ConvertA2AToContent(tt.a2aParts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, parts, tt.want)

			// Verify first part details
			if len(parts) > 0 && len(tt.a2aParts) > 0 {
				if textPart, ok := tt.a2aParts[0].(*protocol.TextPart); ok {
					assert.Equal(t, PartTypeText, parts[0].Type)
					textData := parts[0].Data.(*TextPartData)
					assert.Equal(t, textPart.Text, textData.Text)
				}
			}
		})
	}
}

func TestPartConverter_ConvertContentToA2A(t *testing.T) {
	converter := NewPartConverter()

	tests := []struct {
		name    string
		parts   []*Part
		want    int
		wantErr bool
	}{
		{
			name: "single text part",
			parts: []*Part{
				{
					Type: PartTypeText,
					Data: &TextPartData{Text: "Hello world"},
				},
			},
			want:    1,
			wantErr: false,
		},
		{
			name: "function call part",
			parts: []*Part{
				{
					Type: PartTypeFunctionCall,
					Data: &FunctionCallData{
						Name: "get_weather",
						Args: map[string]interface{}{"city": "London"},
						ID:   "call_123",
					},
				},
			},
			want:    1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a2aParts, err := converter.ConvertContentToA2A(tt.parts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, a2aParts, tt.want)
		})
	}
}

func TestPartConverter_ExtractText(t *testing.T) {
	converter := NewPartConverter()

	parts := []*Part{
		{
			Type: PartTypeText,
			Data: &TextPartData{Text: "Hello "},
		},
		{
			Type: PartTypeText,
			Data: &TextPartData{Text: "world"},
		},
	}

	text := converter.ExtractText(parts)
	assert.Equal(t, "Hello world", text)
}

func TestPartConverter_RoundTrip(t *testing.T) {
	converter := NewPartConverter()

	original := []protocol.Part{
		&protocol.TextPart{Text: "Test message"},
	}

	// A2A -> Content
	contentParts, err := converter.ConvertA2AToContent(original)
	require.NoError(t, err)
	require.Len(t, contentParts, 1)

	// Content -> A2A
	a2aParts, err := converter.ConvertContentToA2A(contentParts)
	require.NoError(t, err)
	require.Len(t, a2aParts, 1)

	// Verify text is preserved
	textPart, ok := a2aParts[0].(*protocol.TextPart)
	require.True(t, ok)
	assert.Equal(t, "Test message", textPart.Text)
}
