package declarative

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	adkoutputschema "github.com/kagent-dev/kagent/go/adk/pkg/outputschema"
)

// maxNesting bounds nested workflow nodes so that a document cannot drive the
// recursive builder arbitrarily deep.
const maxNesting = 8

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

// IsValidation reports whether err is, or wraps, a *ValidationError.
func IsValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// identifier is the shape of agent names and node ids. It matches the names
// ADK accepts for agents and keeps ids usable in event paths.
var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// modelName additionally allows the separators provider model ids use.
var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

// toolName allows the characters that MCP and function tool names use.
var toolName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.:-]*$`)

// Validate checks everything that can be decided from the document alone:
// field shapes, per-class and per-type field rules, and name references
// between agents and nodes. Whether the Go values a document names exist is
// checked by Registry.Validate; graph semantics (reachability, cycles, fan-in)
// are checked by Build, where the ADK workflow engine owns them.
func Validate(spec *Spec) error {
	return validate(spec, nil, nil)
}

// ValidateWithRules is Validate for documents that use node types added with
// Registry.RegisterNodeType. extra rules take precedence over the built-ins.
func ValidateWithRules(spec *Spec, extra map[NodeType]NodeRule) error {
	return validate(spec, extra, nil)
}

func validate(spec *Spec, extra map[NodeType]NodeRule, refs *Registry) error {
	rules := maps.Clone(builtinNodeRules)
	maps.Copy(rules, extra)
	v := &validator{rules: rules, refs: refs}
	v.spec(spec)
	if len(v.issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: v.issues}
}

type validator struct {
	rules map[NodeType]NodeRule
	// refs, when set, resolves the names a document uses to registrations.
	refs   *Registry
	issues []Issue
}

func (v *validator) add(path, format string, args ...any) {
	v.issues = append(v.issues, Issue{Path: path, Message: fmt.Sprintf(format, args...)})
}

// checkFields reports required fields that are unset and set fields that the
// owner accepts neither as required nor as optional.
func (v *validator) checkFields(path, owner string, present map[string]bool, required, optional []string) {
	for _, field := range required {
		if !present[field] {
			v.add(path+"."+field, "required for %s", owner)
		}
	}
	for _, field := range slices.Sorted(maps.Keys(present)) {
		if present[field] && !slices.Contains(required, field) && !slices.Contains(optional, field) {
			v.add(path+"."+field, "not allowed on %s", owner)
		}
	}
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

// agentRule lists the AgentSpec fields a class requires and accepts beyond
// name and agent_class.
type agentRule struct {
	required, optional []string
}

var llmAgentFields = []string{"mode", "global_instruction", "tools", "output_key", "output_schema", "include_contents"}

var agentRules = map[AgentClass]agentRule{
	ClassWorkflow: {required: []string{"nodes"}, optional: []string{"description", "sub_agents", "entry", "edges"}},
	ClassLlmAgent: {
		required: []string{"model", "instruction"},
		optional: append([]string{"description", "sub_agents", "disallow_transfer_to_parent", "disallow_transfer_to_peers"}, llmAgentFields...),
	},
	ClassSequential: {required: []string{"sub_agents"}, optional: []string{"description"}},
	ClassParallel:   {required: []string{"sub_agents"}, optional: []string{"description"}},
	ClassLoop:       {required: []string{"sub_agents"}, optional: []string{"description", "max_iterations"}},
}

func presentAgentFields(a *AgentSpec) map[string]bool {
	present := llmPresent(&a.LLMSpec)
	maps.Copy(present, map[string]bool{
		"description": a.Description != "", "sub_agents": len(a.SubAgents) > 0,
		"entry": a.Entry != "", "nodes": len(a.Nodes) > 0, "edges": len(a.Edges) > 0,
		"disallow_transfer_to_parent": a.DisallowTransferToParent,
		"disallow_transfer_to_peers":  a.DisallowTransferToPeers,
		"max_iterations":              a.MaxIterations != 0,
	})
	return present
}

func llmPresent(l *LLMSpec) map[string]bool {
	return map[string]bool{
		"model": l.Model != "", "mode": l.Mode != "", "instruction": l.Instruction != "",
		"global_instruction": l.GlobalInstruction != "", "tools": len(l.Tools) > 0,
		"output_key": l.OutputKey != "", "output_schema": len(l.OutputSchema) > 0,
		"include_contents": l.IncludeContents != "",
	}
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

	rule, ok := agentRules[a.AgentClass]
	if !ok {
		v.add(path+".agent_class", "must be one of %s, got %q", strings.Join(agentClassNames(), ", "), a.AgentClass)
		return
	}
	v.checkFields(path, "agent_class "+string(a.AgentClass), presentAgentFields(a), rule.required, rule.optional)

	switch a.AgentClass {
	case ClassWorkflow:
		v.graph(path, a.Nodes, a.Edges, a.Entry, classes, 0, true)
	case ClassLlmAgent:
		v.llm(path, &a.LLMSpec)
	}
}

func agentClassNames() []string {
	names := make([]string, 0, len(agentRules))
	for c := range agentRules {
		names = append(names, string(c))
	}
	slices.Sort(names)
	return names
}

// llm validates the fields shared by LlmAgent agents and llm nodes.
func (v *validator) llm(path string, l *LLMSpec) {
	switch l.Mode {
	case "", ModeChat, ModeTask, ModeSingleTurn:
	default:
		v.add(path+".mode", "must be one of chat, task, single_turn, got %q", l.Mode)
	}
	switch l.IncludeContents {
	case "", IncludeContentsDefault, IncludeContentsNone:
	default:
		v.add(path+".include_contents", "must be default or none, got %q", l.IncludeContents)
	}
	if len(l.OutputSchema) > 0 {
		if _, err := adkoutputschema.ToGenAISchema(l.OutputSchema); err != nil {
			v.add(path+".output_schema", "%v", err)
		}
	}
	if l.Model != "" {
		switch {
		case !modelName.MatchString(l.Model):
			v.add(path+".model", "%q must match %s", l.Model, modelName)
		case v.refs != nil && !v.refs.hasModel(l.Model):
			v.add(path+".model", "model %q is not registered", l.Model)
		}
	}
	seen := map[string]bool{}
	for i, t := range l.Tools {
		p := fmt.Sprintf("%s.tools[%d]", path, i)
		switch {
		case seen[t]:
			v.add(p, "duplicate tool %q", t)
		case v.refs != nil && !v.refs.hasToolOrToolset(t):
			v.add(p, "tool %q is not registered", t)
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
	NodeFunction: {Required: []string{"function"}, Optional: []string{"retry", "timeout"}},
	NodeJoin:     {},
	NodeTool:     {Required: []string{"tool"}, Optional: []string{"retry", "timeout"}},
	NodeAgent:    {Required: []string{"agent"}, Optional: []string{"retry", "timeout"}},
	NodeLLM: {
		Required: []string{"model", "instruction"},
		Optional: []string{"mode", "global_instruction", "tools", "output_key", "output_schema", "include_contents", "retry", "timeout"},
	},
	NodeWorkflow:       {Required: []string{"nodes", "edges"}},
	NodeParallelWorker: {Optional: []string{"function", "tool", "agent", "max_concurrency", "retry", "timeout"}},
	NodeDynamic:        {Required: []string{"function"}, Optional: []string{"max_iters", "children", "retry", "timeout"}},
	NodeHITL:           {Required: []string{"message"}},
}

// structuredTerminals are node types whose output is structured data that no
// A2A client displays; a branch must not end on one.
var structuredTerminals = []NodeType{NodeTool, NodeJoin, NodeParallelWorker}

// presentFields names the type-specific fields a node sets.
func presentFields(n *NodeSpec) map[string]bool {
	present := llmPresent(&n.LLMSpec)
	maps.Copy(present, map[string]bool{
		"function": n.Function != "", "tool": n.Tool != "", "agent": n.Agent != "",
		"max_concurrency": n.MaxConcurrency != 0, "max_iters": n.MaxIters != 0,
		"children": len(n.Children) > 0, "message": n.Message != "",
		"nodes": len(n.Nodes) > 0, "edges": len(n.Edges) > 0,
		"retry": n.Retry != nil, "timeout": n.Timeout != "",
	})
	return present
}

// graph validates one node/edge scope: the root of a Workflow agent or the
// body of a nested workflow node. terminal reports whether the scope's own
// terminal nodes end the run, which holds for the root scope and for nested
// scopes whose workflow node is itself terminal.
func (v *validator) graph(path string, nodes []NodeSpec, edges []EdgeSpec, entry string, classes map[string]AgentClass, depth int, terminal bool) {
	if depth > maxNesting {
		v.add(path, "workflow nodes are nested deeper than %d levels", maxNesting)
		return
	}

	ids := make(map[string]bool, len(nodes))
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
	}

	outgoing := make(map[string]bool, len(edges))
	for _, e := range edges {
		outgoing[e.From] = true
	}
	children := v.children(path, nodes, ids)

	agentTargets := make(map[string]string)
	for i := range nodes {
		n := &nodes[i]
		p := fmt.Sprintf("%s.nodes[%d]", path, i)
		v.node(p, n, classes, depth, terminal && !outgoing[n.ID] && !children[n.ID])
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

	v.edges(path, nodes, edges, entry, ids, children)
}

// children validates the children lists of dynamic nodes and returns the set
// of ids that some dynamic node owns.
func (v *validator) children(path string, nodes []NodeSpec, ids map[string]bool) map[string]bool {
	owned := map[string]bool{}
	for i := range nodes {
		n := &nodes[i]
		for j, child := range n.Children {
			p := fmt.Sprintf("%s.nodes[%d].children[%d]", path, i, j)
			switch {
			case !ids[child]:
				v.add(p, "unknown node %q", child)
			case child == n.ID:
				v.add(p, "a node cannot be its own child")
			default:
				owned[child] = true
			}
		}
	}
	return owned
}

func (v *validator) edges(path string, nodes []NodeSpec, edges []EdgeSpec, entry string, ids, children map[string]bool) {
	if entry != "" && !ids[entry] {
		v.add(path+".entry", "unknown node %q", entry)
	}
	hasStart := entry != ""
	connected := map[string]bool{entry: true}
	for child := range children {
		connected[child] = true
	}
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
		for _, end := range []string{e.From, e.To} {
			if children[end] {
				v.add(p, "node %q is a dynamic child and cannot have edges", end)
			}
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

func (v *validator) node(path string, n *NodeSpec, classes map[string]AgentClass, depth int, terminal bool) {
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
	v.checkFields(path, "node type "+string(n.Type), presentFields(n), rule.Required, rule.Optional)

	if terminal && slices.Contains(structuredTerminals, n.Type) {
		v.add(path, "terminal %s node emits structured output that A2A clients do not display; end the branch with a function node", n.Type)
	}

	switch n.Type {
	case NodeFunction:
		v.function(path, n.Function)
	case NodeTool:
		v.toolNode(path, n.Tool)
	case NodeParallelWorker:
		v.parallelWorker(path, n, classes)
	case NodeDynamic:
		if n.MaxIters < 0 {
			v.add(path+".max_iters", "must not be negative")
		}
		if n.Function != "" && v.refs != nil && !v.refs.hasDynamic(n.Function) {
			v.add(path+".function", "dynamic function %q is not registered", n.Function)
		}
	case NodeLLM:
		v.llm(path, &n.LLMSpec)
		if n.Mode != "" && n.Mode != ModeSingleTurn {
			v.add(path+".mode", "an inline llm node runs mid-graph and must be single_turn, got %q", n.Mode)
		}
	case NodeAgent:
		v.agentTarget(path+".agent", n.Agent, classes)
	case NodeWorkflow:
		v.graph(path, n.Nodes, n.Edges, "", classes, depth+1, terminal)
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

func (v *validator) function(path, name string) {
	if name != "" && v.refs != nil && !v.refs.hasFunction(name) {
		v.add(path+".function", "function %q is not registered", name)
	}
}

func (v *validator) toolNode(path, name string) {
	switch {
	case name == "":
	case !toolName.MatchString(name):
		v.add(path+".tool", "%q must match %s", name, toolName)
	case v.refs != nil && !v.refs.hasTool(name):
		v.add(path+".tool", "tool %q is not registered", name)
	}
}

func (v *validator) agentTarget(path, name string, classes map[string]AgentClass) {
	if name == "" {
		return
	}
	class, known := classes[name]
	switch {
	case !known:
		v.add(path, "unknown agent %q", name)
	case class == ClassWorkflow:
		v.add(path, "agent %q is a Workflow; use a workflow node to nest graphs", name)
	}
}

func (v *validator) parallelWorker(path string, n *NodeSpec, classes map[string]AgentClass) {
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
	v.function(path, n.Function)
	v.toolNode(path, n.Tool)
	v.agentTarget(path+".agent", n.Agent, classes)
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
