# Memories

## Patterns

### mem-1771887910-f54e
> MCP server for Kanban API at go/cmd/kanban-mcp/main.go uses modelcontextprotocol/go-sdk v1.3.1. Tool: list_kanban_cards (optional namespace+stage filters). Supports --http addr (streamable HTTP) or stdio (default). Calls kagent via pkg/client/kanban.go KanbanClient → GET /api/kanban. Pattern: mcp.AddTool[In,Out](server, &mcp.Tool{...}, handler); server.Run(ctx, &mcp.StdioTransport{}) for stdio.
<!-- tags: kanban, mcp, go-sdk | created: 2026-02-23 -->

### mem-1771808996-f7e4
> Kanban API pattern: kanban.go handler derives KanbanCard[] from v1alpha2.AgentList using accepted/deploymentReady conditions → Inbox/Assigned/InProgress stages. Route: GET /api/kanban. Types in go/pkg/client/api/types.go. UI calls fetchApi('/kanban').
<!-- tags: kanban, api, agents | created: 2026-02-23 -->

## Decisions

## Fixes

## Context
