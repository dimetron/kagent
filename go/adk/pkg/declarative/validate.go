package declarative

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Issue is one validation failure located by a field path such as
// agents[0].nodes[2].function.
type Issue struct {
	Path    string
	Message string
}

func (i Issue) String() string { return i.Path + ": " + i.Message }

// ValidationError aggregates every issue found in a document so an author
// fixes them in one pass.
type ValidationError struct {
	Issues []Issue
}

func (e *ValidationError) Error() string {
	lines := make([]string, len(e.Issues))
	for i, issue := range e.Issues {
		lines[i] = issue.String()
	}
	return fmt.Sprintf("invalid %s document:\n  %s", KindAgentSystem, strings.Join(lines, "\n  "))
}

// identifier is the shape of agent names and node ids. It matches the names
// ADK accepts for agents and keeps ids usable in event paths.
var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// modelName additionally allows the separators provider model ids use.
var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

// Validate checks everything that can be decided from the document alone:
// field shapes, per-type field rules, and name references. Registry lookups
// and graph semantics (reachability, cycles, fan-in) are checked by Build,
// where the ADK workflow engine owns them.
func Validate(spec *Spec) error {
	return ValidateWithRules(spec, nil)
}

// ValidateWithRules is Validate for documents that use node types added with
// Registry.RegisterNodeType. extra rules take precedence over the built-ins.
func ValidateWithRules(spec *Spec, extra map[NodeType]NodeRule) error {
	rules := maps.Clone(builtinNodeRules)
	maps.Copy(rules, extra)
	v := &validator{rules: rules}
	v.spec(spec)
	if len(v.issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: v.issues}
}

// IsValidation reports whether err is, or wraps, a *ValidationError.
func IsValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

type validator struct {
	rules  map[NodeType]NodeRule
	issues []Issue
}

func (v *validator) add(path, format string, args ...any) {
	v.issues = append(v.issues, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
}

func (v *validator) spec(spec *Spec) {
	if spec == nil {
		v.add("$", "document is required")
		return
	}
	if spec.APIVersion != APIVersion {
		v.add("apiVersion", "must be %q, got %q", APIVersion, spec.APIVersion)
	}
	if spec.Kind != KindAgentSystem {
		v.add("kind", "must be %q, got %q", KindAgentSystem, spec.Kind)
	}
	if len(spec.Agents) == 0 {
		v.add("agents", "at least one agent is required")
		return
	}

	classes := make(map[string]AgentClass, len(spec.Agents))
	for i, a := range spec.Agents {
		path := fmt.Sprintf("agents[%d]", i)
		switch {
		case !identifier.MatchString(a.Name):
			v.add(path+".name", "%q must match %s", a.Name, identifier)
		case a.Name == "user":
			v.add(path+".name", `"user" is reserved for end-user input`)
		default:
			if _, dup := classes[a.Name]; dup {
				v.add(path+".name", "duplicate agent name %q", a.Name)
			}
			classes[a.Name] = a.AgentClass
		}
	}
	for i := range spec.Agents {
		v.agent(fmt.Sprintf("agents[%d]", i), &spec.Agents[i], classes)
	}
	v.agentTree(spec)
}

func (v *validator) agent(path string, a *AgentSpec, classes map[string]AgentClass) {
	for i, name := range a.SubAgents {
		p := fmt.Sprintf("%s.sub_agents[%d]", path, i)
		if _, ok := classes[name]; !ok {
			v.add(p, "unknown agent %q", name)
		} else if name == a.Name {
			v.add(p, "an agent cannot be its own sub-agent")
		}
	}

	switch a.AgentClass {
	case ClassWorkflow:
		v.rejectFields(path, "agent_class Workflow", map[string]bool{
			"model": a.Model != "", "mode": a.Mode != "", "instruction": a.Instruction != "",
			"tools": len(a.Tools) > 0, "output_key": a.OutputKey != "",
		})
		if len(a.Nodes) == 0 {
			v.add(path+".nodes", "a Workflow needs at least one node")
		}
		v.graph(path, a.Nodes, a.Edges, a.Entry, classes)
	case ClassLlmAgent:
		v.rejectFields(path, "agent_class LlmAgent", map[string]bool{
			"entry": a.Entry != "", "nodes": len(a.Nodes) > 0, "edges": len(a.Edges) > 0,
		})
		v.llmFields(path, a.Model, a.Mode, a.Instruction, a.Tools)
	default:
		v.add(path+".agent_class", "must be %q or %q, got %q", ClassWorkflow, ClassLlmAgent, a.AgentClass)
	}
}

// rejectFields reports every set field that the owner does not accept.
func (v *validator) rejectFields(path, owner string, set map[string]bool) {
	for _, field := range sortedKeys(set) {
		if set[field] {
			v.add(path+"."+field, "not allowed on %s", owner)
		}
	}
}

func (v *validator) llmFields(path, model string, mode Mode, instruction string, tools []string) {
	if model == "" {
		v.add(path+".model", "required")
	}
	if instruction == "" {
		v.add(path+".instruction", "required")
	}
	switch mode {
	case "", ModeChat, ModeTask, ModeSingleTurn:
	default:
		v.add(path+".mode", "must be one of chat, task, single_turn, got %q", mode)
	}
	seen := map[string]bool{}
	for i, t := range tools {
		if seen[t] {
			v.add(fmt.Sprintf("%s.tools[%d]", path, i), "duplicate tool %q", t)
		}
		seen[t] = true
	}
}

// NodeRule lists the NodeSpec fields a node type requires and accepts beyond
// the common ones (id, type, description). Field names are the JSON names.
type NodeRule struct {
	Required []string
	Optional []string
}

var builtinNodeRules = map[NodeType]NodeRule{
	NodeFunction:       {Required: []string{"function"}, Optional: []string{"retry", "timeout"}},
	NodeJoin:           {},
	NodeTool:           {Required: []string{"tool"}, Optional: []string{"retry", "timeout"}},
	NodeAgent:          {Required: []string{"agent"}, Optional: []string{"retry", "timeout"}},
	NodeLLM:            {Required: []string{"model", "instruction"}, Optional: []string{"mode", "tools", "output_key", "retry", "timeout"}},
	NodeWorkflow:       {Required: []string{"nodes", "edges"}},
	NodeParallelWorker: {Optional: []string{"function", "tool", "agent", "max_concurrency", "retry", "timeout"}},
	NodeDynamic:        {Required: []string{"function"}, Optional: []string{"max_iters", "retry", "timeout"}},
	NodeHITL:           {Required: []string{"message"}},
}

// presentFields names the type-specific fields a node sets.
func presentFields(n *NodeSpec) map[string]bool {
	return map[string]bool{
		"function": n.Function != "", "tool": n.Tool != "", "agent": n.Agent != "",
		"max_concurrency": n.MaxConcurrency != 0, "max_iters": n.MaxIters != 0,
		"message": n.Message != "", "model": n.Model != "", "mode": n.Mode != "",
		"instruction": n.Instruction != "", "tools": len(n.Tools) > 0, "output_key": n.OutputKey != "",
		"nodes": len(n.Nodes) > 0, "edges": len(n.Edges) > 0,
		"retry": n.Retry != nil, "timeout": n.Timeout != "",
	}
}

// graph validates one node/edge scope: the root of a Workflow agent or the
// body of a nested workflow node.
func (v *validator) graph(path string, nodes []NodeSpec, edges []EdgeSpec, entry string, classes map[string]AgentClass) {
	ids := make(map[string]bool, len(nodes))
	agentTargets := make(map[string]string)
	for i := range nodes {
		n := &nodes[i]
		p := fmt.Sprintf("%s.nodes[%d]", path, i)
		switch {
		case !identifier.MatchString(n.ID):
			v.add(p+".id", "%q must match %s", n.ID, identifier)
		case n.ID == Start:
			v.add(p+".id", "%q is reserved", Start)
		case ids[n.ID]:
			v.add(p+".id", "duplicate node id %q", n.ID)
		default:
			ids[n.ID] = true
		}
		v.node(p, n, classes)
		if n.Agent != "" && (n.Type == NodeAgent || n.Type == NodeParallelWorker) {
			if prev, dup := agentTargets[n.Agent]; dup {
				v.add(p+".agent", "agent %q is already used by node %q in this graph; ADK names an agent node after its agent", n.Agent, prev)
			}
			agentTargets[n.Agent] = n.ID
		}
	}
	for _, n := range nodes {
		if n.Type == NodeAgent && ids[n.Agent] {
			v.add(path+".nodes", "agent %q collides with a node id in the same graph", n.Agent)
		}
	}

	if entry != "" && !ids[entry] {
		v.add(path+".entry", "unknown node %q", entry)
	}
	hasStart := entry != ""
	connected := map[string]bool{entry: true}
	seen := map[EdgeSpec]bool{}
	defaults := map[string]bool{}
	for i, e := range edges {
		p := fmt.Sprintf("%s.edges[%d]", path, i)
		if e.From != Start && !ids[e.From] {
			v.add(p+".from", "unknown node %q", e.From)
		}
		switch {
		case e.To == Start:
			v.add(p+".to", "%q cannot be an edge target", Start)
		case !ids[e.To]:
			v.add(p+".to", "unknown node %q", e.To)
		}
		connected[e.From], connected[e.To] = true, true
		if e.From == Start {
			hasStart = true
			if e.Route != "" {
				v.add(p+".route", "%q does not emit routes", Start)
			}
		}
		if seen[e] {
			v.add(p, "duplicate edge")
		}
		seen[e] = true
		if e.Route == RouteDefault {
			if defaults[e.From] {
				v.add(p+".route", "node %q has more than one %s edge", e.From, RouteDefault)
			}
			defaults[e.From] = true
		}
	}
	for i, n := range nodes {
		if ids[n.ID] && !connected[n.ID] {
			v.add(fmt.Sprintf("%s.nodes[%d]", path, i), "node %q is not connected by any edge", n.ID)
		}
	}
	if !hasStart && len(nodes) > 0 {
		v.add(path+".edges", "no edge from %s and no entry: the graph has no starting node", Start)
	}
}

func (v *validator) node(path string, n *NodeSpec, classes map[string]AgentClass) {
	rule, ok := v.rules[n.Type]
	if !ok {
		types := make([]string, 0, len(v.rules))
		for t := range v.rules {
			types = append(types, string(t))
		}
		slices.Sort(types)
		v.add(path+".type", "unknown node type %q (known: %s)", n.Type, strings.Join(types, ", "))
		return
	}

	present := presentFields(n)
	for _, field := range rule.Required {
		if !present[field] {
			v.add(path+"."+field, "required for node type %s", n.Type)
		}
	}
	allowed := append(slices.Clone(rule.Required), rule.Optional...)
	for _, field := range sortedKeys(present) {
		if present[field] && !slices.Contains(allowed, field) {
			v.add(path+"."+field, "not allowed on node type %s", n.Type)
		}
	}

	switch n.Type {
	case NodeParallelWorker:
		targets := 0
		for _, set := range []bool{n.Function != "", n.Tool != "", n.Agent != ""} {
			if set {
				targets++
			}
		}
		if targets != 1 {
			v.add(path, "parallel_worker needs exactly one of function, tool or agent, got %d", targets)
		}
		if n.MaxConcurrency < 0 {
			v.add(path+".max_concurrency", "must not be negative")
		}
	case NodeDynamic:
		if n.MaxIters < 0 {
			v.add(path+".max_iters", "must not be negative")
		}
	case NodeLLM:
		v.llmFields(path, n.Model, n.Mode, n.Instruction, n.Tools)
		if n.Mode != "" && n.Mode != ModeSingleTurn {
			v.add(path+".mode", "an inline llm node runs mid-graph and must be single_turn, got %q", n.Mode)
		}
	case NodeAgent:
		class, known := classes[n.Agent]
		if !known {
			v.add(path+".agent", "unknown agent %q", n.Agent)
		} else if class == ClassWorkflow {
			v.add(path+".agent", "agent %q is a Workflow; use a workflow node to nest graphs", n.Agent)
		}
	case NodeWorkflow:
		v.graph(path, n.Nodes, n.Edges, "", classes)
	}
	if n.Type == NodeParallelWorker && n.Agent != "" {
		if _, known := classes[n.Agent]; !known {
			v.add(path+".agent", "unknown agent %q", n.Agent)
		}
	}

	if n.Timeout != "" {
		if d, err := time.ParseDuration(n.Timeout); err != nil || d <= 0 {
			v.add(path+".timeout", "must be a positive duration such as 30s, got %q", n.Timeout)
		}
	}
	if n.Retry != nil {
		v.retry(path+".retry", n.Retry)
	}
}

func (v *validator) retry(path string, r *RetrySpec) {
	if r.MaxAttempts < 0 {
		v.add(path+".max_attempts", "must not be negative")
	}
	for field, value := range map[string]string{"initial_delay": r.InitialDelay, "max_delay": r.MaxDelay} {
		if value == "" {
			continue
		}
		if d, err := time.ParseDuration(value); err != nil || d < 0 {
			v.add(path+"."+field, "must be a non-negative duration such as 1s, got %q", value)
		}
	}
	if r.BackoffFactor != nil && *r.BackoffFactor <= 0 {
		v.add(path+".backoff_factor", "must be positive")
	}
	if r.Jitter != nil && (*r.Jitter < 0 || *r.Jitter > 1) {
		v.add(path+".jitter", "must be within [0, 1]")
	}
}

// agentTree checks that sub_agents form a forest: ADK names must be unique
// within an agent tree and a sub-agent has one parent.
func (v *validator) agentTree(spec *Spec) {
	parent := map[string]string{}
	for i, a := range spec.Agents {
		for j, sub := range a.SubAgents {
			p := fmt.Sprintf("agents[%d].sub_agents[%d]", i, j)
			if prev, taken := parent[sub]; taken {
				v.add(p, "agent %q is already a sub-agent of %q; an agent has one parent", sub, prev)
				continue
			}
			parent[sub] = a.Name
		}
	}
	for i, a := range spec.Agents {
		seen := map[string]bool{a.Name: true}
		for cur, ok := parent[a.Name]; ok; cur, ok = parent[cur] {
			if seen[cur] {
				v.add(fmt.Sprintf("agents[%d].sub_agents", i), "agent %q is part of a sub-agent cycle", a.Name)
				break
			}
			seen[cur] = true
		}
	}
}

func sortedKeys(m map[string]bool) []string {
	return slices.Sorted(maps.Keys(m))
}
