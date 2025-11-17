package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
)

// A2AEventPublisher implements EventPublisher for A2A protocol
type A2AEventPublisher struct {
	mu         sync.RWMutex
	subscribers map[string][]chan models.A2AEvent
}

// NewA2AEventPublisher creates a new A2A event publisher
func NewA2AEventPublisher() EventPublisher {
	return &A2AEventPublisher{
		subscribers: make(map[string][]chan models.A2AEvent),
	}
}

// PublishEvent publishes an event to all subscribers
func (p *A2AEventPublisher) PublishEvent(ctx context.Context, event models.A2AEvent) error {
	p.mu.RLock()
	subscribers, exists := p.subscribers[event.TaskID]
	p.mu.RUnlock()

	if !exists || len(subscribers) == 0 {
		// No subscribers, log and continue
		return nil
	}

	// Publish to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- event:
			// Event sent
		default:
			// Channel full, skip
		}
	}

	return nil
}

// Subscribe subscribes to events for a specific task
func (p *A2AEventPublisher) Subscribe(taskID string) <-chan models.A2AEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	ch := make(chan models.A2AEvent, 100)
	p.subscribers[taskID] = append(p.subscribers[taskID], ch)

	return ch
}

// Unsubscribe removes a subscription
func (p *A2AEventPublisher) Unsubscribe(taskID string, ch <-chan models.A2AEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	subscribers := p.subscribers[taskID]
	for i, sub := range subscribers {
		if sub == ch {
			// Remove subscriber
			p.subscribers[taskID] = append(subscribers[:i], subscribers[i+1:]...)
			close(sub)
			break
		}
	}

	// Clean up if no more subscribers
	if len(p.subscribers[taskID]) == 0 {
		delete(p.subscribers, taskID)
	}
}

// ConvertToA2ATaskStatusUpdate converts internal event to A2A protocol event
func ConvertToA2ATaskStatusUpdate(event models.A2AEvent) (*protocol.TaskStatusUpdateEvent, error) {
	statusUpdate := &protocol.TaskStatusUpdateEvent{
		TaskId: event.TaskID,
	}

	// Extract status from event data
	if status, ok := event.Data["status"].(models.TaskStatus); ok {
		statusUpdate.Status = status.ToA2A()
	}

	// Extract message if present
	if message, ok := event.Data["message"].(string); ok {
		statusUpdate.Message = &message
	}

	// Add metadata
	if metadata, ok := event.Data["metadata"].(map[string]interface{}); ok {
		metadataJSON, err := json.Marshal(metadata)
		if err == nil {
			metadataStr := string(metadataJSON)
			statusUpdate.Metadata = &metadataStr
		}
	}

	return statusUpdate, nil
}

// ConvertToA2AArtifactUpdate converts internal event to A2A artifact update
func ConvertToA2AArtifactUpdate(event models.A2AEvent) (*protocol.TaskArtifactUpdateEvent, error) {
	artifactUpdate := &protocol.TaskArtifactUpdateEvent{
		TaskId: event.TaskID,
	}

	// Extract artifact data
	if artifact, ok := event.Data["artifact"].(models.Artifact); ok {
		artifactUpdate.ArtifactId = &artifact.ID
		artifactUpdate.Content = &artifact.Content
		artifactUpdate.ContentType = &artifact.ContentType
	}

	return artifactUpdate, nil
}

// HTTPEventPublisher implements EventPublisher using HTTP callbacks
type HTTPEventPublisher struct {
	webhookURL string
}

// NewHTTPEventPublisher creates a new HTTP event publisher
func NewHTTPEventPublisher(webhookURL string) EventPublisher {
	return &HTTPEventPublisher{
		webhookURL: webhookURL,
	}
}

// PublishEvent publishes an event via HTTP callback
func (p *HTTPEventPublisher) PublishEvent(ctx context.Context, event models.A2AEvent) error {
	// TODO: Implement HTTP webhook callback
	// This would POST the event to the configured webhook URL
	return fmt.Errorf("HTTP event publishing not yet implemented")
}
