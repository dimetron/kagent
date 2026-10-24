package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kagent-dev/kagent/go/adk/pkg/declarative"
)

// TestAgentsDocumentBuilds guards the example document against drift from the
// registry it is served with.
func TestAgentsDocumentBuilds(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key") // building an OpenAI model needs a key; no request is made
	reg, err := newRegistry()
	require.NoError(t, err)
	spec, err := reg.Parse(agentsDocument)
	require.NoError(t, err)
	system, err := declarative.Build(t.Context(), spec, reg)
	require.NoError(t, err)
	require.Equal(t, "support_triage", system.Root().Name())
}
