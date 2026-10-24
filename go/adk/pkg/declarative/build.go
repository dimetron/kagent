package declarative

import (
	"context"
	"fmt"
	"slices"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/agent/workflowagent"
	"google.golang.org/adk/v2/workflow"
)

// System is the set of agents built from one document.
type System struct {
	root   agent.Agent
	agents map[string]agent.Agent
	names  []string
}

// Root is the first agent of the document.
func (s *System) Root() agent.Agent { return s.root }

// Agent returns the agent declared under name.
func (s *System) Agent(name string) (agent.Agent, bool) {
	a, ok := s.agents[name]
	return a, ok
}

// Names lists the declared agents in document order.
func (s *System) Names() []string { return slices.Clone(s.names) }

// Build validates spec against reg and constructs every declared agent. An
// agent that is reachable from several places (sub_agents, agent nodes) is
// built once and shared. Graph semantics such as reachability, unconditional
// cycles and fan-in are enforced by the ADK workflow engine and reported with
// the agent they belong to.
func Build(ctx context.Context, spec *Spec, reg *Registry) (*System, error) {
	if reg == nil {
		reg = NewRegistry()
	}
	if err := reg.Validate(spec); err != nil {
		return nil, err
	}
	b := &builder{
		ctx:   ctx,
		reg:   reg,
		specs: make(map[string]*AgentSpec, len(spec.Agents)),
		built: make(map[string]agent.Agent, len(spec.Agents)),
	}
	sys := &System{agents: b.built}
	for i := range spec.Agents {
		b.specs[spec.Agents[i].Name] = &spec.Agents[i]
		sys.names = append(sys.names, spec.Agents[i].Name)
	}
	for _, name := range sys.names {
		if _, err := b.agent(name); err != nil {
			return nil, err
		}
	}
	sys.root = b.built[sys.names[0]]
	return sys, nil
}

// builder memoizes agents by name. Validation guarantees the references it
// follows are acyclic: sub_agents form a forest and agent nodes only target
// LlmAgents, which never reference a Workflow.
type builder struct {
	ctx   context.Context
	reg   *Registry
	specs map[string]*AgentSpec
	built map[string]agent.Agent
}

func (b *builder) agent(name string) (agent.Agent, error) {
	if a, ok := b.built[name]; ok {
		return a, nil
	}
	spec := b.specs[name]
	var (
		a   agent.Agent
		err error
	)
	switch spec.AgentClass {
	case ClassWorkflow:
		a, err = b.workflowAgent(spec)
	case ClassLlmAgent:
		a, err = b.llmAgent(spec)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build agent %q: %w", name, err)
	}
	b.built[name] = a
	return a, nil
}

func (b *builder) subAgents(names []string) ([]agent.Agent, error) {
	subs := make([]agent.Agent, 0, len(names))
	for _, n := range names {
		a, err := b.agent(n)
		if err != nil {
			return nil, err
		}
		subs = append(subs, a)
	}
	return subs, nil
}

func (b *builder) llmAgent(spec *AgentSpec) (agent.Agent, error) {
	return b.newLLM(spec.Name, spec.Description, spec.Model, spec.Instruction, spec.OutputKey, spec.Mode, spec.Tools, spec.SubAgents)
}

// newLLM is shared by LlmAgent agents and inline llm nodes.
func (b *builder) newLLM(name, description, model, instruction, outputKey string, mode Mode, tools, subAgents []string) (agent.Agent, error) {
	llm, err := b.reg.resolveModel(b.ctx, model)
	if err != nil {
		return nil, err
	}
	toolList, toolsets, err := b.reg.resolveTools(tools)
	if err != nil {
		return nil, err
	}
	subs, err := b.subAgents(subAgents)
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        name,
		Description: description,
		Model:       llm,
		Instruction: instruction,
		Tools:       toolList,
		Toolsets:    toolsets,
		SubAgents:   subs,
		OutputKey:   outputKey,
		Mode:        llmagent.Mode(mode),
	})
}

func (b *builder) workflowAgent(spec *AgentSpec) (agent.Agent, error) {
	nodes, err := b.nodes(spec.Nodes)
	if err != nil {
		return nil, err
	}
	edges := toEdges(nodes, spec.Edges, spec.Entry)
	subs, err := b.subAgents(spec.SubAgents)
	if err != nil {
		return nil, err
	}
	return workflowagent.New(workflowagent.Config{
		Name:        spec.Name,
		Description: spec.Description,
		SubAgents:   subs,
		Edges:       edges,
	})
}

// nodes builds every node of one graph scope, keyed by document id.
func (b *builder) nodes(specs []NodeSpec) (map[string]workflow.Node, error) {
	bc := &BuildContext{Context: b.ctx, Registry: b.reg, Agent: b.agent}
	nodes := make(map[string]workflow.Node, len(specs))
	for i := range specs {
		n := &specs[i]
		node, err := b.node(bc, n)
		if err != nil {
			return nil, fmt.Errorf("failed to build node %q: %w", n.ID, err)
		}
		nodes[n.ID] = node
	}
	return nodes, nil
}

// toEdges resolves document edges against built nodes. entry is sugar for a
// START edge and is skipped when the edge already exists.
func toEdges(nodes map[string]workflow.Node, specs []EdgeSpec, entry string) []workflow.Edge {
	resolve := func(id string) workflow.Node {
		if id == Start {
			return workflow.Start
		}
		return nodes[id]
	}
	edges := make([]workflow.Edge, 0, len(specs)+1)
	hasEntry := false
	for _, e := range specs {
		edge := workflow.Edge{From: resolve(e.From), To: resolve(e.To)}
		switch e.Route {
		case "":
		case RouteDefault:
			edge.Route = workflow.Default
		default:
			edge.Route = workflow.StringRoute(e.Route)
		}
		hasEntry = hasEntry || (e.From == Start && e.To == entry && e.Route == "")
		edges = append(edges, edge)
	}
	if entry != "" && !hasEntry {
		edges = append([]workflow.Edge{{From: workflow.Start, To: nodes[entry]}}, edges...)
	}
	return edges
}
