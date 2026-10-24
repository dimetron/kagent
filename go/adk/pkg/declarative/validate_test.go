package declarative

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const docHeader = "apiVersion: adk.declarative/v0.2\nkind: AgentSystem\nagents:\n"

// wf wraps nodes and edges in a minimal Workflow agent.
func wf(nodes, edges string) string {
	return docHeader + "  - name: wf\n    agent_class: Workflow\n    nodes: " + nodes + "\n    edges: " + edges + "\n"
}

func TestShowcaseIsValid(t *testing.T) {
	data, err := os.ReadFile("testdata/showcase.yaml")
	require.NoError(t, err)
	spec, err := Parse(data)
	require.NoError(t, err)
	require.Len(t, spec.Agents, 5)
}

func TestValidateReportsFieldPaths(t *testing.T) {
	startTo := func(id string) string { return "[{ from: START, to: " + id + " }]" }
	tests := []struct {
		name string
		doc  string
		want []string
	}{
		{"wrong apiVersion and kind", "apiVersion: v1\nkind: Pod\nagents:\n  - { name: a, agent_class: LlmAgent, model: m, instruction: i }\n",
			[]string{`apiVersion: must be "adk.declarative/v0.2"`, `kind: must be "AgentSystem"`}},
		{"no agents", "apiVersion: adk.declarative/v0.2\nkind: AgentSystem\nagents: []\n",
			[]string{"agents: at least one agent is required"}},
		{"bad and reserved agent names", docHeader + "  - { name: 'a b', agent_class: LlmAgent, model: m, instruction: i }\n  - { name: user, agent_class: LlmAgent, model: m, instruction: i }\n",
			[]string{"agents[0].name:", `agents[1].name: "user" is reserved`}},
		{"duplicate agent", docHeader + "  - { name: a, agent_class: LlmAgent, model: m, instruction: i }\n  - { name: a, agent_class: LlmAgent, model: m, instruction: i }\n",
			[]string{`agents[1].name: duplicate agent name "a"`}},
		{"unknown agent class", docHeader + "  - { name: a, agent_class: Sequential }\n",
			[]string{"agents[0].agent_class: must be"}},
		{"llm fields on a workflow", docHeader + "  - { name: a, agent_class: Workflow, model: m, instruction: i, nodes: [{ id: n, type: join }], edges: [{ from: START, to: n }] }\n",
			[]string{"agents[0].instruction: not allowed on agent_class Workflow", "agents[0].model: not allowed on agent_class Workflow"}},
		{"workflow fields on an llm agent", docHeader + "  - { name: a, agent_class: LlmAgent, model: m, instruction: i, entry: x }\n",
			[]string{"agents[0].entry: not allowed on agent_class LlmAgent"}},
		{"llm agent requires model and instruction", docHeader + "  - { name: a, agent_class: LlmAgent }\n",
			[]string{"agents[0].model: required", "agents[0].instruction: required"}},
		{"bad mode and duplicate tool", docHeader + "  - { name: a, agent_class: LlmAgent, model: m, instruction: i, mode: weird, tools: [t, t] }\n",
			[]string{"agents[0].mode: must be one of", "agents[0].tools[1]: duplicate tool"}},
		{"unknown and self sub-agent", docHeader + "  - { name: a, agent_class: LlmAgent, model: m, instruction: i, sub_agents: [a, ghost] }\n",
			[]string{"sub_agents[0]: an agent cannot be its own sub-agent", `sub_agents[1]: unknown agent "ghost"`}},
		{"sub-agent with two parents", docHeader +
			"  - { name: a, agent_class: LlmAgent, model: m, instruction: i, sub_agents: [c] }\n" +
			"  - { name: b, agent_class: LlmAgent, model: m, instruction: i, sub_agents: [c] }\n" +
			"  - { name: c, agent_class: LlmAgent, model: m, instruction: i }\n",
			[]string{`agents[1].sub_agents[0]: agent "c" is already a sub-agent of "a"`}},
		{"sub-agent cycle", docHeader +
			"  - { name: a, agent_class: LlmAgent, model: m, instruction: i, sub_agents: [b] }\n" +
			"  - { name: b, agent_class: LlmAgent, model: m, instruction: i, sub_agents: [a] }\n",
			[]string{"is part of a sub-agent cycle"}},
		{"workflow without nodes", docHeader + "  - { name: a, agent_class: Workflow }\n",
			[]string{"agents[0].nodes: a Workflow needs at least one node"}},
		{"bad node ids", wf("[{ id: START, type: join }, { id: 'a-b', type: join }, { id: x, type: join }, { id: x, type: join }]", "[{ from: START, to: x }]"),
			[]string{`nodes[0].id: "START" is reserved`, "nodes[1].id:", `nodes[3].id: duplicate node id "x"`}},
		{"unknown node type", wf("[{ id: n, type: teleport }]", startTo("n")),
			[]string{`nodes[0].type: unknown node type "teleport" (known: `}},
		{"function node requires function", wf("[{ id: n, type: function }]", startTo("n")),
			[]string{"nodes[0].function: required for node type function"}},
		{"field not allowed on node type", wf("[{ id: n, type: join, function: f }]", startTo("n")),
			[]string{"nodes[0].function: not allowed on node type join"}},
		{"retry not allowed on hitl", wf("[{ id: n, type: hitl, message: m, retry: { max_attempts: 2 } }]", startTo("n")),
			[]string{"nodes[0].retry: not allowed on node type hitl"}},
		{"parallel worker needs exactly one target", wf("[{ id: n, type: parallel_worker, function: f, tool: t }, { id: m, type: parallel_worker }]", "[{ from: START, to: n }, { from: n, to: m }]"),
			[]string{"nodes[0]: parallel_worker needs exactly one of function, tool or agent, got 2", "nodes[1]: parallel_worker needs exactly one of function, tool or agent, got 0"}},
		{"negative bounds", wf("[{ id: n, type: parallel_worker, function: f, max_concurrency: -1 }, { id: d, type: dynamic, function: f, max_iters: -1 }]", "[{ from: START, to: n }, { from: n, to: d }]"),
			[]string{"nodes[0].max_concurrency: must not be negative", "nodes[1].max_iters: must not be negative"}},
		{"inline llm must be single_turn", wf("[{ id: n, type: llm, model: m, instruction: i, mode: chat }]", startTo("n")),
			[]string{`nodes[0].mode: an inline llm node runs mid-graph and must be single_turn, got "chat"`}},
		{"agent node targets", docHeader +
			"  - { name: llm, agent_class: LlmAgent, model: m, instruction: i }\n" +
			"  - name: wf\n    agent_class: Workflow\n    nodes: [{ id: a, type: agent, agent: ghost }, { id: b, type: agent, agent: wf }, { id: c, type: agent, agent: llm }, { id: d, type: agent, agent: llm }]\n" +
			"    edges: [{ from: START, to: a }, { from: a, to: b }, { from: b, to: c }, { from: c, to: d }]\n",
			[]string{`nodes[0].agent: unknown agent "ghost"`, `nodes[1].agent: agent "wf" is a Workflow`, `nodes[3].agent: agent "llm" is already used by node "c"`}},
		{"agent name collides with node id", docHeader +
			"  - { name: llm, agent_class: LlmAgent, model: m, instruction: i }\n" +
			"  - name: wf\n    agent_class: Workflow\n    nodes: [{ id: llm, type: join }, { id: a, type: agent, agent: llm }]\n    edges: [{ from: START, to: llm }, { from: llm, to: a }]\n",
			[]string{`agent "llm" collides with a node id`}},
		{"bad durations", wf("[{ id: n, type: function, function: f, timeout: soon, retry: { initial_delay: later, max_delay: -1s } }]", startTo("n")),
			[]string{"nodes[0].timeout: must be a positive duration", "retry.initial_delay: must be a non-negative duration", "retry.max_delay: must be a non-negative duration"}},
		{"bad retry numbers", wf("[{ id: n, type: function, function: f, retry: { max_attempts: -1, backoff_factor: 0, jitter: 2 } }]", startTo("n")),
			[]string{"retry.max_attempts: must not be negative", "retry.backoff_factor: must be positive", "retry.jitter: must be within [0, 1]"}},
		{"edge endpoints", wf("[{ id: n, type: join }]", "[{ from: ghost, to: n }, { from: START, to: START }, { from: n, to: ghost }]"),
			[]string{`edges[0].from: unknown node "ghost"`, `edges[1].to: "START" cannot be an edge target`, `edges[2].to: unknown node "ghost"`}},
		{"START cannot route", wf("[{ id: n, type: join }]", "[{ from: START, to: n, route: X }]"),
			[]string{`edges[0].route: "START" does not emit routes`}},
		{"duplicate edge and default", wf("[{ id: a, type: join }, { id: b, type: join }, { id: c, type: join }]",
			"[{ from: START, to: a }, { from: START, to: a }, { from: a, to: b, route: DEFAULT }, { from: a, to: c, route: DEFAULT }]"),
			[]string{"edges[1]: duplicate edge", `edges[3].route: node "a" has more than one DEFAULT edge`}},
		{"graph without a start", wf("[{ id: a, type: join }, { id: b, type: join }]", "[{ from: a, to: b }]"),
			[]string{"no edge from START and no entry"}},
		{"node outside every edge", wf("[{ id: a, type: join }, { id: b, type: join }]", "[{ from: START, to: a }]"),
			[]string{`nodes[1]: node "b" is not connected by any edge`}},
		{"unknown entry", wf("[{ id: a, type: join }]", "[{ from: START, to: a }]") + "    entry: ghost\n",
			[]string{`entry: unknown node "ghost"`}},
		{"errors inside a nested workflow carry the full path", wf("[{ id: outer, type: workflow, nodes: [{ id: in, type: function }], edges: [{ from: START, to: in }] }]", startTo("outer")),
			[]string{"agents[0].nodes[0].nodes[0].function: required for node type function"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.doc))
			require.Error(t, err)
			require.True(t, IsValidation(err))
			for _, want := range tt.want {
				require.Contains(t, err.Error(), want)
			}
		})
	}
}

func TestValidateAggregatesAllIssues(t *testing.T) {
	_, err := Parse([]byte("apiVersion: nope\nkind: nope\nagents:\n  - { name: a, agent_class: LlmAgent }\n"))
	var ve *ValidationError
	require.ErrorAs(t, err, &ve)
	require.GreaterOrEqual(t, len(ve.Issues), 4)
	require.True(t, strings.HasPrefix(err.Error(), "invalid AgentSystem document:"))
}

func TestValidateNilSpec(t *testing.T) {
	require.ErrorContains(t, Validate(nil), "document is required")
}

func TestEntryMayRepeatAnExplicitStartEdge(t *testing.T) {
	doc := wf("[{ id: a, type: join }]", "[{ from: START, to: a }]") + "    entry: a\n"
	_, err := Parse([]byte(doc))
	require.NoError(t, err)
}

func TestParseRejectsUnknownFields(t *testing.T) {
	tests := map[string]string{
		"agent":  docHeader + "  - { name: a, agent_class: LlmAgent, model: m, instruction: i, temprature: 1 }\n",
		"node":   wf("[{ id: n, type: join, colour: red }]", "[{ from: START, to: n }]"),
		"edge":   wf("[{ id: n, type: join }]", "[{ from: START, to: n, weight: 2 }]"),
		"retry":  wf("[{ id: n, type: function, function: f, retry: { attempts: 2 } }]", "[{ from: START, to: n }]"),
		"syntax": "agents: [",
	}
	for name, doc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(doc))
			require.ErrorContains(t, err, "failed to decode AgentSystem document")
			require.False(t, IsValidation(err))
		})
	}
}
