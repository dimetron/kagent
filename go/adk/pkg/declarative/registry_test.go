package declarative

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
	"google.golang.org/adk/v2/workflow"
)

func noopFunc(agent.Context, any) (Result, error) { return Result{}, nil }

func TestRegistryRejectsBadRegistrations(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterFunction("f", noopFunc))
	require.ErrorContains(t, reg.RegisterFunction("f", noopFunc), "already registered")
	require.ErrorContains(t, reg.RegisterFunction("bad name", noopFunc), "must match")
	require.ErrorContains(t, reg.RegisterFunction("nilfn", nil), "nil func")

	require.ErrorContains(t, reg.RegisterDynamic("d", nil), "nil func")
	require.ErrorContains(t, reg.RegisterModel("stub", NewStubModel("stub")), "already registered")
	require.ErrorContains(t, reg.RegisterModel("m", nil), "nil model")
	require.ErrorContains(t, reg.RegisterModel("not valid!", NewStubModel("x")), "must match")
	require.NoError(t, reg.RegisterModel("openai/gpt-4o:latest", NewStubModel("gpt")))

	require.ErrorContains(t, reg.RegisterTool(nil), "nil tool")
	require.ErrorContains(t, reg.RegisterToolset(nil), "nil toolset")
	require.ErrorContains(t, reg.RegisterTool(reg.tools["load_memory"]), "already registered")
}

func TestRegistryToolAndToolsetNamesAreDisjoint(t *testing.T) {
	reg := NewRegistry()
	ts := namedToolset("shared")
	require.NoError(t, reg.RegisterToolset(ts))
	clash, err := functiontool.New(functiontool.Config{Name: "shared", Description: "d"},
		func(agent.Context, struct{}) (string, error) { return "", nil })
	require.NoError(t, err)
	require.ErrorContains(t, reg.RegisterTool(clash), "used by another registration")
}

type namedToolset string

func (n namedToolset) Name() string                                     { return string(n) }
func (n namedToolset) Tools(agent.ReadonlyContext) ([]tool.Tool, error) { return nil, nil }

func TestRegistryBuiltinTools(t *testing.T) {
	reg := NewRegistry()
	tools, toolsets, err := reg.resolveTools([]string{"preload_memory", "load_memory"})
	require.NoError(t, err)
	require.Len(t, tools, 2)
	require.Empty(t, toolsets)
	_, _, err = reg.resolveTools([]string{"ghost"})
	require.ErrorContains(t, err, `tool "ghost" is not registered`)
}

func TestRegistryToolsetsResolve(t *testing.T) {
	reg := NewRegistry()
	require.NoError(t, reg.RegisterToolset(namedToolset("mcp_server")))
	tools, toolsets, err := reg.resolveTools([]string{"mcp_server", "load_memory"})
	require.NoError(t, err)
	require.Len(t, tools, 1)
	require.Len(t, toolsets, 1)
}

func TestRegisterNodeTypeRules(t *testing.T) {
	reg := NewRegistry()
	build := func(*BuildContext, *NodeSpec) (workflow.Node, error) { return nil, nil }
	require.ErrorContains(t, reg.RegisterNodeType(NodeJoin, NodeRule{}, build), "built-in types cannot be replaced")
	require.ErrorContains(t, reg.RegisterNodeType("bad-name", NodeRule{}, build), "must match")
	require.ErrorContains(t, reg.RegisterNodeType("custom", NodeRule{}, nil), "nil builder")
	require.NoError(t, reg.RegisterNodeType("custom", NodeRule{}, build))
	require.ErrorContains(t, reg.RegisterNodeType("custom", NodeRule{}, build), "already registered")
}

func TestDynamicEnvBudget(t *testing.T) {
	child := workflow.NewFunctionNode("c", func(agent.Context, any) (string, error) { return "ok", nil }, workflow.NodeConfig{})
	bounded := &DynamicEnv{MaxIters: 1, runs: 1}
	_, err := bounded.Run(child, nil)
	require.ErrorIs(t, err, ErrIterationBudget)
	require.Equal(t, 1, bounded.Runs(), "a refused run is not counted")
}

func TestResultHelpers(t *testing.T) {
	require.Equal(t, Result{Output: 1}, Output(1))
	require.Equal(t, Result{Output: 1, Routes: []string{"a", "b"}}, Route(1, "a", "b"))
}
