package converters

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	apperrors "github.com/kagent-dev/kagent/go/pkg/adk/errors"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// PartConverter handles conversion between A2A Parts and generic Content Parts
type PartConverter struct{}

// NewPartConverter creates a new PartConverter
func NewPartConverter() *PartConverter {
	return &PartConverter{}
}

// ConvertA2AToContent converts A2A message parts to Content parts
func (c *PartConverter) ConvertA2AToContent(a2aParts []protocol.Part) ([]*Part, error) {
	var parts []*Part

	for _, a2aPart := range a2aParts {
		part, err := c.convertA2APart(a2aPart)
		if err != nil {
			return nil, err
		}
		if part != nil {
			parts = append(parts, part)
		}
	}

	return parts, nil
}

func (c *PartConverter) convertA2APart(a2aPart protocol.Part) (*Part, error) {
	switch p := a2aPart.(type) {
	case *protocol.TextPart:
		return &Part{
			Type: PartTypeText,
			Data: &TextPartData{
				Text: p.Text,
			},
		}, nil

	case *protocol.FilePart:
		fileData := &FilePartData{
			MimeType: p.MimeType,
		}

		// Handle URI or inline data
		if p.FileURI != "" {
			fileData.URI = p.FileURI
		} else if len(p.FileData) > 0 {
			fileData.Data = p.FileData
		}

		return &Part{
			Type: PartTypeFile,
			Data: fileData,
		}, nil

	case *protocol.DataPart:
		// DataPart can contain base64 encoded data
		data, err := base64.StdEncoding.DecodeString(p.Data)
		if err != nil {
			// If not base64, treat as raw data
			data = []byte(p.Data)
		}

		return &Part{
			Type: PartTypeFile,
			Data: &FilePartData{
				MimeType: p.MimeType,
				Data:     data,
			},
		}, nil

	default:
		// Unknown part type - try to handle as JSON
		jsonData, err := json.Marshal(a2aPart)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrCodeConversion,
				fmt.Sprintf("unsupported part type: %T", a2aPart), nil)
		}

		return &Part{
			Type: PartTypeText,
			Data: &TextPartData{
				Text: string(jsonData),
			},
		}, nil
	}
}

// ConvertContentToA2A converts Content parts back to A2A parts
func (c *PartConverter) ConvertContentToA2A(parts []*Part) ([]protocol.Part, error) {
	var a2aParts []protocol.Part

	for _, part := range parts {
		a2aPart, err := c.convertContentPart(part)
		if err != nil {
			return nil, err
		}
		if a2aPart != nil {
			a2aParts = append(a2aParts, a2aPart)
		}
	}

	return a2aParts, nil
}

func (c *PartConverter) convertContentPart(part *Part) (protocol.Part, error) {
	switch part.Type {
	case PartTypeText:
		data, ok := part.Data.(*TextPartData)
		if !ok {
			// Try to convert from map
			if dataMap, ok := part.Data.(map[string]interface{}); ok {
				if text, ok := dataMap["text"].(string); ok {
					return &protocol.TextPart{
						Text: text,
					}, nil
				}
			}
			return nil, apperrors.New(apperrors.ErrCodeConversion, "invalid text part data", nil)
		}

		return &protocol.TextPart{
			Text: data.Text,
		}, nil

	case PartTypeFile:
		data, ok := part.Data.(*FilePartData)
		if !ok {
			// Try to convert from map
			if dataMap, ok := part.Data.(map[string]interface{}); ok {
				fileData := &FilePartData{}
				if uri, ok := dataMap["uri"].(string); ok {
					fileData.URI = uri
				}
				if mimeType, ok := dataMap["mime_type"].(string); ok {
					fileData.MimeType = mimeType
				}
				if dataBytes, ok := dataMap["data"].([]byte); ok {
					fileData.Data = dataBytes
				}
				data = fileData
			} else {
				return nil, apperrors.New(apperrors.ErrCodeConversion, "invalid file part data", nil)
			}
		}

		if data.URI != "" {
			return &protocol.FilePart{
				FileURI:  data.URI,
				MimeType: data.MimeType,
			}, nil
		}

		return &protocol.FilePart{
			FileData: data.Data,
			MimeType: data.MimeType,
		}, nil

	case PartTypeFunctionCall:
		data, ok := part.Data.(*FunctionCallData)
		if !ok {
			return nil, apperrors.New(apperrors.ErrCodeConversion, "invalid function call data", nil)
		}

		// Convert to text representation for A2A
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrCodeConversion, "failed to marshal function call", err)
		}

		return &protocol.TextPart{
			Text: fmt.Sprintf("[Function Call: %s]\n%s", data.Name, string(jsonData)),
		}, nil

	case PartTypeFunctionResponse:
		data, ok := part.Data.(*FunctionResponseData)
		if !ok {
			return nil, apperrors.New(apperrors.ErrCodeConversion, "invalid function response data", nil)
		}

		return &protocol.TextPart{
			Text: fmt.Sprintf("[Function Response: %s]\n%s", data.Name, data.Response),
		}, nil

	case PartTypeCodeExecution:
		data, ok := part.Data.(*CodeExecutionData)
		if !ok {
			return nil, apperrors.New(apperrors.ErrCodeConversion, "invalid code execution data", nil)
		}

		output := data.Output
		if data.Error != "" {
			output = fmt.Sprintf("Error: %s\n%s", data.Error, output)
		}

		return &protocol.TextPart{
			Text: output,
		}, nil

	default:
		// Unknown type - convert to text
		jsonData, err := json.Marshal(part.Data)
		if err != nil {
			return nil, apperrors.New(apperrors.ErrCodeConversion,
				fmt.Sprintf("failed to convert unknown part type: %s", part.Type), err)
		}

		return &protocol.TextPart{
			Text: string(jsonData),
		}, nil
	}
}

// ExtractText extracts all text from content parts
func (c *PartConverter) ExtractText(parts []*Part) string {
	var text string
	for _, part := range parts {
		if part.Type == PartTypeText {
			if data, ok := part.Data.(*TextPartData); ok {
				text += data.Text
			}
		}
	}
	return text
}
