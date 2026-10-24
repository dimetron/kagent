package declarative

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/agent"
	adkmodel "google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/workflow"
	"google.golang.org/genai"
)

func userMessage(text string) *genai.Content { return genai.NewContentFromText(text, genai.RoleUser) }

// outputs returns the non-nil event outputs in arrival order.
func outputs(events []*session.Event) []any {
	var out []any
	for _, ev := range events {
		if ev.Output != nil {
			out = append(out, ev.Output)
		}
	}
	return out
}

func requestedInput(events []*session.Event) *session.RequestInput {
	for _, ev := range events {
		if ev.RequestedInput != nil {
			return ev.RequestedInput
		}
	}
	return nil
}

func buildShowcase(t *testing.T) *System {
	t.Helper()
	sys, err := Build(t.Context(), loadShowcase(t), testRegistry(t))
	require.NoError(t, err)
	return sys
}

func TestShowcaseRoutesEachBranch(t *testing.T) {
	sys := buildShowcase(t)
	tests := []struct {
		name  string
		input string
		final any
	}{
		{"BUG runs parallel worker then nested workflow", "bug x y",
			"summary:exec(plan([item:bug item:x item:y]))"},
		{"BILLING runs the tool node", "bill me",
			"summary:map[customer:c-42 plan:pro]"},
		{"SUPPORT runs the agent node", "help me", "summary:[stub] help me"},
		{"unmatched route takes DEFAULT", "zzz", "item:zzz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, id := newSession(t)
			got := outputs(run(t, sys.Root(), userMessage(tt.input), svc, id))
			require.Equal(t, tt.final, got[len(got)-1])
		})
	}
}

func TestShowcaseFanOutJoinsEveryBranch(t *testing.T) {
	svc, id := newSession(t)
	got := outputs(run(t, buildShowcase(t).Root(), userMessage("zzz"), svc, id))
	require.Contains(t, got, map[string]any{"branch_a": "a", "branch_b": "b", "branch_c": "c"})
}

func TestShowcaseOtherPausesForApprovalAndResumes(t *testing.T) {
	sys := buildShowcase(t)
	svc, id := newSession(t)

	paused := run(t, sys.Root(), userMessage("other thing"), svc, id)
	req := requestedInput(paused)
	require.NotNil(t, req, "hitl node must request input")
	require.Equal(t, "Confirm final reply before send (yes/no)", req.Message)
	// The dynamic node spent exactly max_iters child runs before the pause.
	require.Contains(t, outputs(paused), "[stub] other thing after 2 runs")
	for _, o := range outputs(paused) {
		require.NotContains(t, fmt.Sprint(o), "summary:", "terminal must not run before approval")
	}

	reply := &genai.Content{Role: genai.RoleUser, Parts: []*genai.Part{{FunctionResponse: &genai.FunctionResponse{
		ID: req.InterruptID, Name: workflow.WorkflowInputFunctionCallName,
		Response: map[string]any{"response": "yes"},
	}}}}
	resumed := run(t, sys.Root(), reply, svc, id)
	got := outputs(resumed)
	require.Equal(t, "summary:yes", got[len(got)-1])
}

func TestSystemAccessors(t *testing.T) {
	sys := buildShowcase(t)
	require.Equal(t, "multi_agent_root", sys.Root().Name())
	require.Equal(t, []string{"multi_agent_root", "specialist_helper", "code_expert", "learning_coordinator", "docs_expert"}, sys.Names())

	coordinator, ok := sys.Agent("learning_coordinator")
	require.True(t, ok)
	require.Equal(t, "docs_expert", coordinator.SubAgents()[0].Name())
	_, ok = sys.Agent("missing")
	require.False(t, ok)

	// An agent used by sub_agents and by an agent node is one shared instance.
	helper, _ := sys.Agent("specialist_helper")
	require.Same(t, helper, sys.Root().SubAgents()[0])
}

func TestBuildNilRegistryUsesDefaults(t *testing.T) {
	spec, err := Parse([]byte(`
apiVersion: adk.declarative/v0.2
kind: AgentSystem
agents:
  - { name: solo, agent_class: LlmAgent, model: stub, instruction: hi }
`))
	require.NoError(t, err)
	sys, err := Build(t.Context(), spec, nil)
	require.NoError(t, err)
	svc, id := newSession(t)
	require.Contains(t, outputsOrText(run(t, sys.Root(), userMessage("ping"), svc, id)), "[stub] ping")
}

func outputsOrText(events []*session.Event) []string {
	var out []string
	for _, ev := range events {
		if ev.Content != nil {
			for _, p := range ev.Content.Parts {
				if p != nil && p.Text != "" {
					out = append(out, p.Text)
				}
			}
		}
	}
	return out
}

func TestRetryPolicyRerunsFailingFunction(t *testing.T) {
	var calls atomic.Int32
	reg := NewRegistry()
	require.NoError(t, reg.RegisterFunction("flaky", func(agent.Context, any) (Result, error) {
		if calls.Add(1) < 3 {
			return Result{}, errors.New("transient")
		}
		return Output("recovered"), nil
	}))
	spec, err := reg.Parse([]byte(`
apiVersion: adk.declarative/v0.2
kind: AgentSystem
agents:
  - name: wf
    agent_class: Workflow
    nodes:
      - id: only
        type: function
        function: flaky
        retry: { max_attempts: 3, initial_delay: 1ms, max_delay: 2ms, backoff_factor: 1, jitter: 0 }
    edges: [{ from: START, to: only }]
`))
	require.NoError(t, err)
	sys, err := Build(t.Context(), spec, reg)
	require.NoError(t, err)

	svc, id := newSession(t)
	require.Equal(t, []any{"recovered"}, outputs(run(t, sys.Root(), userMessage("go"), svc, id)))
	require.EqualValues(t, 3, calls.Load())
}

func TestBuildFailures(t *testing.T) {
	const header = "apiVersion: adk.declarative/v0.2\nkind: AgentSystem\nagents:\n"
	tests := []struct {
		name string
		doc  string
		want string
	}{
		{"unregistered function", header + `
  - name: wf
    agent_class: Workflow
    nodes: [{ id: n, type: function, function: nope }]
    edges: [{ from: START, to: n }]`, `function "nope" is not registered`},
		{"unregistered tool", header + `
  - name: wf
    agent_class: Workflow
    nodes: [{ id: n, type: tool, tool: nope }]
    edges: [{ from: START, to: n }]`, `tool "nope" is not registered`},
		{"unregistered dynamic function", header + `
  - name: wf
    agent_class: Workflow
    nodes: [{ id: n, type: dynamic, function: nope }]
    edges: [{ from: START, to: n }]`, `dynamic function "nope" is not registered`},
		{"unregistered model", header + `
  - { name: a, agent_class: LlmAgent, model: gpt-nope, instruction: hi }`, `model "gpt-nope" is not registered`},
		{"unregistered llm tool", header + `
  - { name: a, agent_class: LlmAgent, model: stub, instruction: hi, tools: [nope] }`, `tool "nope" is not registered`},
		{"unconditional cycle is rejected by the engine", header + `
  - name: wf
    agent_class: Workflow
    nodes:
      - { id: a, type: function, function: prefix }
      - { id: b, type: function, function: prefix }
    edges:
      - { from: START, to: a }
      - { from: a, to: b }
      - { from: b, to: a }`, `agent "wf"`},
		{"fan-in to a non-join node is rejected by the engine", header + `
  - name: wf
    agent_class: Workflow
    nodes:
      - { id: a, type: function, function: prefix }
      - { id: b, type: function, function: prefix }
      - { id: c, type: function, function: prefix }
    edges:
      - { from: START, to: a }
      - { from: START, to: b }
      - { from: a, to: c }
      - { from: b, to: c }`, `agent "wf"`},
		{"task mode agent cannot be a graph node", header + `
  - { name: worker, agent_class: LlmAgent, model: stub, instruction: hi, mode: task }
  - name: wf
    agent_class: Workflow
    nodes: [{ id: n, type: agent, agent: worker }]
    edges: [{ from: START, to: n }]`, `agent "wf"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := testRegistry(t)
			spec, err := reg.Parse([]byte(tt.doc))
			require.NoError(t, err, "document must be valid; the failure belongs to Build")
			_, err = Build(t.Context(), spec, reg)
			require.ErrorContains(t, err, tt.want)
			require.False(t, IsValidation(err), "build failures are not document validation failures")
		})
	}
}

func TestBuildRejectsInvalidDocuments(t *testing.T) {
	_, err := Build(t.Context(), &Spec{}, NewRegistry())
	require.True(t, IsValidation(err))
}

func TestModelResolverIsConsultedAfterRegisteredModels(t *testing.T) {
	reg := NewRegistry()
	var asked []string
	reg.SetModelResolver(func(_ context.Context, name string) (adkmodel.LLM, error) {
		asked = append(asked, name)
		if name == "forbidden" {
			return nil, errors.New("denied")
		}
		return NewStubModel(name), nil
	})
	build := func(model string) error {
		spec, err := reg.Parse([]byte(fmt.Sprintf(`
apiVersion: adk.declarative/v0.2
kind: AgentSystem
agents:
  - { name: a, agent_class: LlmAgent, model: %q, instruction: hi }
`, model)))
		require.NoError(t, err)
		_, err = Build(t.Context(), spec, reg)
		return err
	}
	require.NoError(t, build("stub"))
	require.Empty(t, asked, "registered models win over the resolver")
	require.NoError(t, build("gpt-4o"))
	require.ErrorContains(t, build("forbidden"), "denied")
	require.Equal(t, []string{"gpt-4o", "forbidden"}, asked)
}

func TestCustomNodeType(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterNodeType("shout", NodeRule{Required: []string{"message"}},
		func(_ *BuildContext, spec *NodeSpec) (workflow.Node, error) {
			return workflow.NewFunctionNode(spec.ID, func(_ agent.Context, in string) (string, error) {
				return in + spec.Message, nil
			}, workflow.NodeConfig{}), nil
		}))
	doc := []byte(`
apiVersion: adk.declarative/v0.2
kind: AgentSystem
agents:
  - name: wf
    agent_class: Workflow
    nodes: [{ id: s, type: shout, message: "!" }]
    edges: [{ from: START, to: s }]
`)
	_, err := Parse(doc)
	require.ErrorContains(t, err, `unknown node type "shout"`, "plain Parse does not know custom types")

	spec, err := reg.Parse(doc)
	require.NoError(t, err)
	sys, err := Build(t.Context(), spec, reg)
	require.NoError(t, err)
	svc, id := newSession(t)
	require.Equal(t, []any{"hey!"}, outputs(run(t, sys.Root(), userMessage("hey"), svc, id)))

	bad, err := reg.Parse([]byte(`
apiVersion: adk.declarative/v0.2
kind: AgentSystem
agents:
  - { name: wf, agent_class: Workflow, nodes: [{ id: s, type: shout }], edges: [{ from: START, to: s }] }
`))
	require.Nil(t, bad)
	require.ErrorContains(t, err, "agents[0].nodes[0].message: required for node type shout")
}

func TestParallelWorkerWrapsToolAndAgent(t *testing.T) {
	reg := testRegistry(t)
	spec, err := reg.Parse([]byte(`
apiVersion: adk.declarative/v0.2
kind: AgentSystem
agents:
  - name: fanout
    agent_class: Workflow
    nodes:
      - { id: split, type: function, function: split_words }
      - { id: lookups, type: parallel_worker, tool: lookup_customer }
    edges:
      - { from: START, to: split }
      - { from: split, to: lookups }
`))
	require.NoError(t, err, "split_words is checked by Build, not by validation")
	require.NoError(t, reg.RegisterFunction("split_words", func(agent.Context, any) (Result, error) {
		return Output([]any{map[string]any{"customer_id": "c-1"}, map[string]any{"customer_id": "c-2"}}), nil
	}))
	sys, err := Build(t.Context(), spec, reg)
	require.NoError(t, err)
	svc, id := newSession(t)
	got := outputs(run(t, sys.Root(), userMessage("go"), svc, id))
	require.Equal(t, []any{
		map[string]any{"customer": "c-1", "plan": "pro"},
		map[string]any{"customer": "c-2", "plan": "pro"},
	}, got[len(got)-1])
}

func TestStubModel(t *testing.T) {
	m := NewStubModel("stub-x")
	require.Equal(t, "stub-x", m.Name())
	next, stop := iter.Pull2(m.GenerateContent(t.Context(), &adkmodel.LLMRequest{Contents: []*genai.Content{
		genai.NewContentFromText("first", genai.RoleUser),
		genai.NewContentFromText("reply", genai.RoleModel),
		genai.NewContentFromText("last", genai.RoleUser),
	}}, false))
	defer stop()
	resp, err, ok := next()
	require.True(t, ok)
	require.NoError(t, err)
	require.Equal(t, "[stub-x] last", resp.Content.Parts[0].Text)
	require.True(t, resp.TurnComplete)

	next2, stop2 := iter.Pull2(m.GenerateContent(t.Context(), nil, false))
	defer stop2()
	resp, _, _ = next2()
	require.Equal(t, "[stub-x] ", resp.Content.Parts[0].Text)
}
