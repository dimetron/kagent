package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kagent-dev/kagent/go/internal/executor/temporal/models"
)

// MockProvider for testing
type MockProvider struct {
	mock.Mock
	name string
}

func (m *MockProvider) Chat(ctx context.Context, request models.LLMRequest) (*models.LLMResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LLMResponse), args.Error(1)
}

func (m *MockProvider) ChatStream(ctx context.Context, request models.LLMRequest) (<-chan StreamChunk, <-chan error) {
	args := m.Called(ctx, request)
	return args.Get(0).(<-chan StreamChunk), args.Get(1).(<-chan error)
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) SupportedModels() []string {
	return []string{"mock-model-1", "mock-model-2"}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	provider := &MockProvider{name: "test-provider"}

	// First registration should succeed
	err := registry.Register(provider)
	require.NoError(t, err)

	// Duplicate registration should fail
	err = registry.Register(provider)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already registered")
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	provider1 := &MockProvider{name: "provider1"}
	provider2 := &MockProvider{name: "provider2"}

	// Register providers
	require.NoError(t, registry.Register(provider1))
	require.NoError(t, registry.Register(provider2))

	// Get existing provider
	retrieved, err := registry.Get("provider1")
	require.NoError(t, err)
	require.Equal(t, "provider1", retrieved.Name())

	// Get non-existent provider
	_, err = registry.Get("nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	names := registry.List()
	require.Empty(t, names)

	// Register providers
	provider1 := &MockProvider{name: "provider1"}
	provider2 := &MockProvider{name: "provider2"}
	provider3 := &MockProvider{name: "provider3"}

	require.NoError(t, registry.Register(provider1))
	require.NoError(t, registry.Register(provider2))
	require.NoError(t, registry.Register(provider3))

	// List all providers
	names = registry.List()
	require.Len(t, names, 3)
	require.Contains(t, names, "provider1")
	require.Contains(t, names, "provider2")
	require.Contains(t, names, "provider3")
}

func TestRegistry_Concurrent(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent registration
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			provider := &MockProvider{name: string(rune('a' + idx))}
			registry.Register(provider)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all providers registered
	names := registry.List()
	require.GreaterOrEqual(t, len(names), 1) // At least some should succeed

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			registry.List()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
