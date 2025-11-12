package a2a

import (
	"context"
	"time"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"trpc.group/trpc-go/trpc-a2a-go/client"
	"trpc.group/trpc-go/trpc-a2a-go/protocol"
	"trpc.group/trpc-go/trpc-a2a-go/taskmanager"
)

const (
	// KeepAliveInterval defines how often to send keep-alive events
	// if no events are received from the agent
	KeepAliveInterval = 30 * time.Second
)

// KeepAliveManager wraps a PassthroughManager and injects keep-alive events
// during long-running agent tasks to prevent connection timeouts
type KeepAliveManager struct {
	*PassthroughManager
}

// NewKeepAliveManager creates a new KeepAliveManager that wraps the client
func NewKeepAliveManager(client *client.A2AClient) taskmanager.TaskManager {
	return &KeepAliveManager{
		PassthroughManager: NewPassthroughManager(client).(*PassthroughManager),
	}
}

// OnSendMessageStream wraps the streaming with keep-alive injection
func (m *KeepAliveManager) OnSendMessageStream(ctx context.Context, request protocol.SendMessageParams) (<-chan protocol.StreamingMessageEvent, error) {
	log := ctrllog.FromContext(ctx).WithName("keepalive-manager")

	// Set message defaults
	if request.Message.MessageID == "" {
		request.Message.MessageID = protocol.GenerateMessageID()
	}
	if request.Message.Kind == "" {
		request.Message.Kind = protocol.KindMessage
	}

	// Get the original channel from the agent
	agentChan, err := m.PassthroughManager.client.StreamMessage(ctx, request)
	if err != nil {
		return nil, err
	}

	// Create a new channel with keep-alive injection
	outputChan := make(chan protocol.StreamingMessageEvent)

	go func() {
		defer close(outputChan)

		ticker := time.NewTicker(KeepAliveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Context cancelled, stop streaming
				log.V(1).Info("Context cancelled, stopping keep-alive manager")
				return

			case event, ok := <-agentChan:
				if !ok {
					// Agent channel closed, we're done
					log.V(1).Info("Agent channel closed")
					return
				}

				// Forward the agent event to output
				select {
				case outputChan <- event:
					// Event sent successfully, reset ticker
					ticker.Reset(KeepAliveInterval)
				case <-ctx.Done():
					return
				}

			case <-ticker.C:
				// No event from agent for KeepAliveInterval, send keep-alive
				log.V(1).Info("Injecting keep-alive event")

				// Create a keep-alive message
				keepAliveMessage := protocol.Message{
					Kind:      "system",
					Role:      "system",
					MessageID: protocol.GenerateMessageID(),
					Parts:     []protocol.Part{protocol.NewTextPart("Keep-alive from server")},
				}

				keepAliveEvent := protocol.StreamingMessageEvent{
					Result: &protocol.TaskStatusUpdateEvent{
						Status: protocol.TaskStatus{
							Message: &keepAliveMessage,
						},
					},
				}

				// Try to send keep-alive
				select {
				case outputChan <- keepAliveEvent:
					log.V(1).Info("Keep-alive event sent")
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputChan, nil
}

// OnResubscribe wraps the resubscribe with keep-alive injection
func (m *KeepAliveManager) OnResubscribe(ctx context.Context, params protocol.TaskIDParams) (<-chan protocol.StreamingMessageEvent, error) {
	log := ctrllog.FromContext(ctx).WithName("keepalive-manager")

	// Get the original channel from resubscribe
	agentChan, err := m.PassthroughManager.client.ResubscribeTask(ctx, params)
	if err != nil {
		return nil, err
	}

	// Create a new channel with keep-alive injection
	outputChan := make(chan protocol.StreamingMessageEvent)

	go func() {
		defer close(outputChan)

		ticker := time.NewTicker(KeepAliveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.V(1).Info("Context cancelled, stopping keep-alive manager (resubscribe)")
				return

			case event, ok := <-agentChan:
				if !ok {
					log.V(1).Info("Agent channel closed (resubscribe)")
					return
				}

				// Forward the agent event
				select {
				case outputChan <- event:
					ticker.Reset(KeepAliveInterval)
				case <-ctx.Done():
					return
				}

			case <-ticker.C:
				log.V(1).Info("Injecting keep-alive event (resubscribe)")

				// Create a keep-alive message
				keepAliveMessage := protocol.Message{
					Kind:      "system",
					Role:      "system",
					MessageID: protocol.GenerateMessageID(),
					Parts:     []protocol.Part{protocol.NewTextPart("Keep-alive from server (resubscribe)")},
				}

				keepAliveEvent := protocol.StreamingMessageEvent{
					Result: &protocol.TaskStatusUpdateEvent{
						Status: protocol.TaskStatus{
							Message: &keepAliveMessage,
						},
					},
				}

				select {
				case outputChan <- keepAliveEvent:
					log.V(1).Info("Keep-alive event sent (resubscribe)")
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputChan, nil
}
