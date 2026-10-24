package declarative

import (
	"fmt"
	"time"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/workflow"
)

// node dispatches on the node type. Registered types are consulted after the
// built-ins, which they cannot shadow.
func (b *builder) node(bc *BuildContext, n *NodeSpec) (workflow.Node, error) {
	cfg, err := nodeConfig(n)
	if err != nil {
		return nil, err
	}
	switch n.Type {
	case NodeFunction:
		fn, err := b.function(n.Function)
		if err != nil {
			return nil, err
		}
		return workflow.NewEmittingFunctionNode(n.ID, routedFunc(fn), cfg), nil
	case NodeJoin:
		return workflow.NewJoinNode(n.ID), nil
	case NodeTool:
		return b.toolNode(n.ID, n.Tool, cfg)
	case NodeAgent:
		return b.agentNode(n.Agent, cfg)
	case NodeLLM:
		return b.llmNode(n, cfg)
	case NodeWorkflow:
		return b.workflowNode(n)
	case NodeParallelWorker:
		return b.parallelWorker(n, cfg)
	case NodeDynamic:
		return b.dynamicNode(n, cfg)
	case NodeHITL:
		return hitlNode(n, cfg), nil
	}
	custom, ok := b.reg.nodeTypes[n.Type]
	if !ok {
		return nil, fmt.Errorf("node type %q is not registered", n.Type)
	}
	return custom.build(bc, n)
}

func (b *builder) function(name string) (Func, error) {
	fn, ok := b.reg.funcs[name]
	if !ok {
		return nil, fmt.Errorf("function %q is not registered", name)
	}
	return fn, nil
}

// routedFunc adapts a Func to the engine. A routed result is emitted as one
// event that carries both the output and the routes; the engine selects edges
// from Event.Routes.
func routedFunc(fn Func) workflow.EmittingFunctionFn[any, any] {
	return func(ctx agent.Context, in any, emit func(*session.Event) error) (any, error) {
		res, err := fn(ctx, in)
		if err != nil {
			return nil, err
		}
		if len(res.Routes) == 0 {
			return res.Output, nil
		}
		ev := session.NewEvent(ctx, ctx.InvocationID())
		ev.Routes = res.Routes
		ev.Output = res.Output
		return nil, emit(ev)
	}
}

func (b *builder) toolNode(id, toolName string, cfg workflow.NodeConfig) (workflow.Node, error) {
	t, ok := b.reg.tools[toolName]
	if !ok {
		return nil, fmt.Errorf("tool %q is not registered", toolName)
	}
	return workflow.NewNamedToolNode(id, t, cfg)
}

// agentNode wraps a document agent. ADK names an agent node after its agent,
// so the document id of such a node is not visible to the engine.
func (b *builder) agentNode(agentName string, cfg workflow.NodeConfig) (workflow.Node, error) {
	a, err := b.agent(agentName)
	if err != nil {
		return nil, err
	}
	return workflow.NewAgentNode(a, cfg)
}

// llmNode builds an inline llm node as an anonymous single-turn agent named
// after the node id.
func (b *builder) llmNode(n *NodeSpec, cfg workflow.NodeConfig) (workflow.Node, error) {
	mode := n.Mode
	if mode == "" {
		mode = ModeSingleTurn
	}
	a, err := b.newLLM(n.ID, n.Description, n.Model, n.Instruction, n.OutputKey, mode, n.Tools, nil)
	if err != nil {
		return nil, err
	}
	return workflow.NewAgentNode(a, cfg)
}

func (b *builder) workflowNode(n *NodeSpec) (workflow.Node, error) {
	nodes, err := b.nodes(n.Nodes)
	if err != nil {
		return nil, err
	}
	return workflow.NewWorkflowNode(n.ID, toEdges(nodes, n.Edges, ""))
}

// parallelWorker fans the input list out over one wrapped node. The wrapped
// node carries no retry policy: ADK applies the worker's own policy per item.
func (b *builder) parallelWorker(n *NodeSpec, cfg workflow.NodeConfig) (workflow.Node, error) {
	var wrapped workflow.Node
	var err error
	switch {
	case n.Function != "":
		var fn Func
		if fn, err = b.function(n.Function); err == nil {
			wrapped = workflow.NewFunctionNode(n.ID+"_item", plainFunc(fn), workflow.NodeConfig{})
		}
	case n.Tool != "":
		wrapped, err = b.toolNode(n.ID+"_item", n.Tool, workflow.NodeConfig{})
	default:
		wrapped, err = b.agentNode(n.Agent, workflow.NodeConfig{})
	}
	if err != nil {
		return nil, err
	}
	return workflow.NewParallelWorker(n.ID, wrapped, n.MaxConcurrency, cfg)
}

// plainFunc adapts a Func whose routes are ignored: a worker item has no
// outgoing edges to route over.
func plainFunc(fn Func) func(agent.Context, any) (any, error) {
	return func(ctx agent.Context, in any) (any, error) {
		res, err := fn(ctx, in)
		return res.Output, err
	}
}

func (b *builder) dynamicNode(n *NodeSpec, cfg workflow.NodeConfig) (workflow.Node, error) {
	fn, ok := b.reg.dynamics[n.Function]
	if !ok {
		return nil, fmt.Errorf("dynamic function %q is not registered", n.Function)
	}
	maxIters := n.MaxIters
	return workflow.NewDynamicNode(n.ID, func(ctx agent.Context, in any, emit func(*session.Event) error) (any, error) {
		return fn(ctx, in, &DynamicEnv{MaxIters: maxIters, Emit: emit, ctx: ctx})
	}, cfg), nil
}

// hitlNode pauses the workflow until a human answers and forwards the answer.
// The node re-runs on resume, so the interrupt id must be stable within a run
// and unique across runs; the invocation id gives both.
func hitlNode(n *NodeSpec, cfg workflow.NodeConfig) workflow.Node {
	rerun := true
	cfg.RerunOnResume = &rerun
	return workflow.NewEmittingFunctionNode(n.ID, func(ctx agent.Context, _ any, emit func(*session.Event) error) (any, error) {
		return workflow.ResumeOrRequestInput(ctx, emit, session.RequestInput{
			InterruptID: n.ID + "-" + ctx.InvocationID(),
			Message:     n.Message,
		})
	}, cfg)
}

// nodeConfig translates the common retry and timeout fields. Validate has
// already checked the syntax of every duration.
func nodeConfig(n *NodeSpec) (workflow.NodeConfig, error) {
	var cfg workflow.NodeConfig
	if n.Timeout != "" {
		d, err := time.ParseDuration(n.Timeout)
		if err != nil {
			return cfg, fmt.Errorf("failed to parse timeout: %w", err)
		}
		cfg.Timeout = d
	}
	if n.Retry != nil {
		rc := workflow.DefaultRetryConfig()
		{
			r := n.Retry
			if r.MaxAttempts != 0 {
				rc.MaxAttempts = r.MaxAttempts
			}
			for _, f := range []struct {
				in  string
				out *time.Duration
			}{{r.InitialDelay, &rc.InitialDelay}, {r.MaxDelay, &rc.MaxDelay}} {
				if f.in == "" {
					continue
				}
				d, err := time.ParseDuration(f.in)
				if err != nil {
					return cfg, fmt.Errorf("failed to parse retry delay: %w", err)
				}
				*f.out = d
			}
			if r.BackoffFactor != nil {
				rc.BackoffFactor = *r.BackoffFactor
			}
			if r.Jitter != nil {
				rc.Jitter = *r.Jitter
			}
		}
		cfg.RetryConfig = rc
	}
	return cfg, nil
}
