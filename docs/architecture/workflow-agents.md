# Declarative Workflow Agents

The Go ADK builds agents from an `adk.declarative/v0.2` `AgentSystem` document
with `go/adk/pkg/declarative`. A Workflow agent is a native ADK 2.x graph
(`workflow` package) executed through `workflowagent`; the package adds no
execution engine of its own. Routing, fan-out and join, retries, timeouts,
pause and resume, telemetry, and A2A streaming are the ADK engine's behavior.

This is a Go ADK library surface. It is not a Kubernetes API field and not a
session or task model: A2A still owns interaction state, and a paused workflow
resumes through the same A2A task as any other ADK agent.

## Document model

A document lists agents; the first is the root. Every agent has an
`agent_class`:

| Class      | Meaning                                                              |
| ---------- | -------------------------------------------------------------------- |
| `Workflow` | A graph of `nodes` and `edges`, optionally with an `entry` shorthand |
| `LlmAgent` | A model-backed agent with `model`, `instruction`, `tools`, `mode`    |

`sub_agents` names other agents of the document. Names are agent identities in
the ADK agent tree, so an agent has at most one parent and `sub_agents` never
cycle. An agent may additionally be the target of any number of `agent` nodes
in different graphs; it is built once and shared.

### Node types

| Type              | Required                   | Behavior                                                        |
| ----------------- | -------------------------- | --------------------------------------------------------------- |
| `function`        | `function`                 | Runs a registered `Func`; may select routes                     |
| `join`            |                            | Fan-in barrier; emits a map keyed by predecessor                |
| `tool`            | `tool`                     | Runs a registered tool with the predecessor output as arguments |
| `agent`           | `agent`                    | Runs an `LlmAgent` of the document                              |
| `llm`             | `model`, `instruction`     | Inline single-turn `LlmAgent` named after the node id           |
| `workflow`        | `nodes`, `edges`           | Nested graph with its own `START`                               |
| `parallel_worker` | one of function/tool/agent | Runs the target once per item of a list input                   |
| `dynamic`         | `function`                 | Registered `DynamicFunc` that runs children under `max_iters`   |
| `hitl`            | `message`                  | Pauses for a human reply and forwards the reply                 |

`function`, `tool`, `agent`, `llm`, `parallel_worker` and `dynamic` nodes accept
`timeout` and `retry`, which map to `workflow.NodeConfig`. Unset retry fields
take the ADK defaults.

Edges connect node ids. `START` is the entry source. An edge may carry a
`route`, matched against the routes a `function` node returns; `DEFAULT` is the
edge taken when no route matched. Unknown fields anywhere in the document are
decode errors.

## Boundaries

| Concern                                                | Owner                          |
| ------------------------------------------------------ | ------------------------------ |
| Document shape, per-type field rules, name references  | `Validate` (pure, no registry) |
| Names that resolve to Go values                        | `Registry`                     |
| Reachability, unconditional cycles, fan-in, task mode  | ADK workflow engine, at `Build` |
| Execution, routing, retries, pause and resume          | ADK workflow engine            |

`Validate` reports every issue with a field path such as
`agents[0].nodes[2].function`. Build failures from the ADK engine are wrapped
with the agent and node they belong to; they are not `ValidationError`s.

A document carries no code. `Registry` attaches behavior by name:
`RegisterFunction`, `RegisterDynamic`, `RegisterTool`, `RegisterToolset`,
`RegisterModel`, and `SetModelResolver` for model names that are not registered
by value (for example to reuse the runtime's provider factories). The offline
`stub` model and the ADK memory tools are preregistered.

## Extending

`Registry.RegisterNodeType` adds a node type: a `NodeRule` listing the
`NodeSpec` fields it requires and accepts, and a `NodeBuilder` returning any
`workflow.Node`. Documents using custom types are parsed with `Registry.Parse`
so validation applies the same rules. Built-in types cannot be replaced.

## Constraints inherited from the ADK engine

- An unset `mode` is `single_turn` for an agent placed as a graph node and a
  declared `chat` mode is honored there, keeping history. `task` agents cannot be
  graph nodes; `Build` surfaces the engine's rejection. Inline `llm` nodes are
  always `single_turn` and validation rejects other modes.
- Mutually exclusive routes must not fan in to a non-`join` node; give each
  route its own terminal. A `join` suits fan-out that always runs every branch.
- An `agent` node is named after its agent inside the engine, so two nodes of
  one graph cannot wrap the same agent and an agent name cannot equal a node id.
- `workflowagent.Config` exposes no engine options, so a concurrency cap for the
  whole graph is not available; `parallel_worker.max_concurrency` bounds one
  worker.
- A `parallel_worker` target runs without routes; a `Func` result's routes are
  ignored there.
- A `hitl` node re-runs on resume. Its interrupt id combines the node id and the
  invocation id, so it is stable within a run and unique across runs.

## Example

`go/adk/examples/declarative` serves a document through the A2A application
builder, with Go functions, a tool, and an OpenAI model resolver registered.
`go/adk/pkg/declarative/testdata/showcase.yaml` exercises every node type and
route form and is the document the package tests run end to end.
