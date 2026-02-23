# Scratchpad

## 2026-02-23 - Add Controller API for Kanban UI

### Understanding

Objective: Add a dedicated `/api/kanban` endpoint in the Go controller HTTP server, then update the UI to use it.

**Current state:**
- Kanban UI exists (`ui/src/app/kanban/page.tsx`, `ui/src/components/kanban/`)
- `ui/src/app/actions/kanban.ts` calls `getAgents()` and maps agents → kanban cards in the UI layer
- Go backend has full agent CRUD at `/api/agents`
- Go has `handlers/handlers.go` pattern: handler type + Base + registration in `server.go`

**Plan:**
1. Add `KanbanCard`, `KanbanStage` types to `go/pkg/client/api/types.go`
2. Add `go/internal/httpserver/handlers/kanban.go` — KanbanHandler deriving cards from agent list
3. Register `/api/kanban` route in `go/internal/httpserver/server.go`
4. Update `ui/src/app/actions/kanban.ts` to call `/kanban` instead of `getAgents()`

**Stage mapping (same logic as UI currently):**
- !accepted && !deploymentReady → Inbox
- accepted && !deploymentReady → Assigned
- accepted && deploymentReady → InProgress

**Priority field:** default "normal" (no automated source)

### Iteration 1 - Add types + handler + route
Pick task: Add Go backend kanban API (types + handler + route)

### Iteration 2 - Update UI
Pick task: Update UI kanban action to call new `/api/kanban` endpoint
