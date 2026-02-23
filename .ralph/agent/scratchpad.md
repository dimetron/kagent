## 2026-02-24 - MCP Server for Kanban API

### Understanding
- Objective: Create MCP Server for Kanban API using modelcontextprotocol/go-sdk
- Existing Kanban API: GET /api/kanban → returns []KanbanCard
- KanbanCard: {id, title, stage, agentName, namespace, priority, createdAt, updatedAt}
- KanbanStage: Inbox, Assigned, InProgress, Review, Done
- Mapping: !accepted&&!deploymentReady→Inbox; accepted&&!deploymentReady→Assigned; both→InProgress
- Existing pkg/client pattern: each resource has a client file, registered in clientset.go
- go-sdk template is at go/cli/internal/mcp/frameworks/golang/templates/
- go/go.mod uses mark3labs/mcp-go (NOT go-sdk); need to add modelcontextprotocol/go-sdk

### Plan
1. Add modelcontextprotocol/go-sdk to go/go.mod (task 1)
2. Create go/pkg/client/kanban.go (task 2)
3. Update go/pkg/client/clientset.go to add Kanban (task 3 - blocked by 2)
4. Create go/cmd/kanban-mcp/main.go - MCP server with list_kanban_cards tool (task 4 - blocked by 2)

### Architecture Decision (confidence 85 - proceed autonomously)
- Standalone binary in go/cmd/kanban-mcp/main.go
- HTTP client approach (call kagent API) vs direct kube client
- HTTP client is more portable, no k8s dependency, simpler
- Server accepts --kagent-url flag or KAGENT_URL env var
- Supports both stdio and HTTP transport (--http flag)

### Implementation Notes
- go-sdk version to use: v0.2.0 (from templates)
- Tool: list_kanban_cards with optional namespace+stage filter params
