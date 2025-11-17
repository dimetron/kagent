package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

type vertexAIProvider struct {
	projectID  string
	location   string
	apiKey     string
	httpClient *http.Client
}

// NewVertexAIProvider creates a new Vertex AI provider
func NewVertexAIProvider(projectID, location, apiKey string) Provider {
	return &vertexAIProvider{
		projectID: projectID,
		location:  location,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (p *vertexAIProvider) Name() string {
	return "vertexai"
}

func (p *vertexAIProvider) SupportedModels() []string {
	return []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-1.0-pro",
		"gemini-1.0-pro-vision",
		"claude-3-5-sonnet@20241022",  // Anthropic via Vertex
		"claude-3-opus@20240229",       // Anthropic via Vertex
		"claude-3-haiku@20240307",      // Anthropic via Vertex
	}
}

func (p *vertexAIProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	// Determine if this is a Gemini or Claude model
	if strings.HasPrefix(request.ModelConfig.Model, "gemini") {
		return p.chatGemini(ctx, request)
	} else if strings.HasPrefix(request.ModelConfig.Model, "claude") {
		return p.chatClaude(ctx, request)
	}

	return nil, fmt.Errorf("unsupported model: %s", request.ModelConfig.Model)
}

func (p *vertexAIProvider) chatGemini(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	// Build Vertex AI Gemini API request
	endpoint := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		p.location, p.projectID, p.location, request.ModelConfig.Model,
	)

	// Convert messages to Gemini format
	contents := make([]map[string]interface{}, 0)
	var systemInstruction string

	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemInstruction = msg.Content
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		content := map[string]interface{}{
			"role": role,
			"parts": []map[string]interface{}{
				{"text": msg.Content},
			},
		}
		contents = append(contents, content)
	}

	// Build request body
	reqBody := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":    request.ModelConfig.Temperature,
			"maxOutputTokens": request.ModelConfig.MaxTokens,
		},
	}

	if systemInstruction != "" {
		reqBody["systemInstruction"] = map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": systemInstruction},
			},
		}
	}

	if len(request.Tools) > 0 {
		tools := p.convertToolsToGemini(request.Tools)
		reqBody["tools"] = tools
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string                 `json:"text,omitempty"`
					FunctionCall map[string]interface{} `json:"functionCall,omitempty"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := geminiResp.Candidates[0]

	// Build response
	response := &models.LLMResponse{
		ModelUsed:    request.ModelConfig.Model,
		FinishReason: strings.ToLower(candidate.FinishReason),
		TokenUsage: models.TokenUsage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		},
	}

	// Extract content and function calls
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			response.Content += part.Text
		}
		if part.FunctionCall != nil {
			// Convert function call to tool call
			name := part.FunctionCall["name"].(string)
			args := part.FunctionCall["args"].(map[string]interface{})
			response.ToolCalls = append(response.ToolCalls, models.ToolCall{
				ID:        fmt.Sprintf("call_%d", len(response.ToolCalls)),
				Name:      name,
				Arguments: args,
				Status:    "pending",
			})
		}
	}

	return response, nil
}

func (p *vertexAIProvider) chatClaude(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	// Build Vertex AI Claude API request (Anthropic via Vertex)
	endpoint := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:rawPredict",
		p.location, p.projectID, p.location, request.ModelConfig.Model,
	)

	// Convert to Anthropic format (similar to anthropic.go)
	messages := make([]map[string]interface{}, 0)
	var systemMessage string

	for _, msg := range request.Messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
			continue
		}

		msgMap := map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		messages = append(messages, msgMap)
	}

	reqBody := map[string]interface{}{
		"anthropic_version": "vertex-2023-10-16",
		"messages":          messages,
		"max_tokens":        request.ModelConfig.MaxTokens,
		"temperature":       request.ModelConfig.Temperature,
	}

	if systemMessage != "" {
		reqBody["system"] = systemMessage
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Execute request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse Claude response
	var claudeResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build response
	response := &models.LLMResponse{
		ModelUsed:    request.ModelConfig.Model,
		FinishReason: claudeResp.StopReason,
		TokenUsage: models.TokenUsage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
	}

	for _, content := range claudeResp.Content {
		if content.Type == "text" {
			response.Content += content.Text
		}
	}

	return response, nil
}

func (p *vertexAIProvider) ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// For now, fall back to non-streaming
		response, err := p.Chat(ctx, request)
		if err != nil {
			errChan <- err
			return
		}

		// Send as single chunk
		chunkChan <- StreamChunk{
			Content:      response.Content,
			ToolCalls:    response.ToolCalls,
			FinishReason: response.FinishReason,
			Delta:        false,
		}
	}()

	return chunkChan, errChan
}

func (p *vertexAIProvider) convertToolsToGemini(tools []models.Tool) []map[string]interface{} {
	geminiTools := make([]map[string]interface{}, 0)

	functionDeclarations := make([]map[string]interface{}, 0)

	for _, tool := range tools {
		functionDeclarations = append(functionDeclarations, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
		})
	}

	if len(functionDeclarations) > 0 {
		geminiTools = append(geminiTools, map[string]interface{}{
			"functionDeclarations": functionDeclarations,
		})
	}

	return geminiTools
}
