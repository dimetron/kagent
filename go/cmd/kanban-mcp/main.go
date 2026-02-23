// Package main implements a Model Context Protocol (MCP) server that exposes
// the kagent Kanban API as MCP tools.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/kagent-dev/kagent/go/pkg/client"
	"github.com/kagent-dev/kagent/go/pkg/client/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	httpAddr  = flag.String("http", "", "if set, serve MCP over streamable HTTP on this address (e.g. :8085); default is stdio")
	kagentURL = flag.String("kagent-url", "", "kagent HTTP server base URL (default: $KAGENT_URL or http://localhost:8083)")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func resolveKagentURL() string {
	if *kagentURL != "" {
		return *kagentURL
	}
	if v := os.Getenv("KAGENT_URL"); v != "" {
		return v
	}
	return "http://localhost:8083"
}

// listKanbanCardsInput defines the input schema for the list_kanban_cards tool.
type listKanbanCardsInput struct {
	// Namespace filters cards by agent namespace; empty means all namespaces.
	Namespace string `json:"namespace,omitempty" description:"Filter cards by agent namespace. Empty or omitted returns cards from all namespaces."`
	// Stage filters cards by kanban stage; empty means all stages.
	Stage string `json:"stage,omitempty" description:"Filter cards by stage: Inbox, Assigned, InProgress, Review, Done. Empty or omitted returns all stages."`
}

// listKanbanCardsOutput is the structured output of the list_kanban_cards tool.
type listKanbanCardsOutput struct {
	Cards []api.KanbanCard `json:"cards"`
	Total int              `json:"total"`
}

func run() error {
	baseURL := resolveKagentURL()
	kagentClient := client.New(baseURL)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "kagent-kanban",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_kanban_cards",
		Description: "List kanban cards derived from kagent Agents. Each card represents an Agent with a stage: Inbox (not accepted), Assigned (accepted, not ready), or InProgress (accepted and deployment ready).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input listKanbanCardsInput) (*mcp.CallToolResult, listKanbanCardsOutput, error) {
		resp, err := kagentClient.Kanban.ListKanbanCards(ctx)
		if err != nil {
			return nil, listKanbanCardsOutput{}, fmt.Errorf("failed to list kanban cards: %w", err)
		}

		cards := resp.Data
		if cards == nil {
			cards = []api.KanbanCard{}
		}

		// Apply optional filters
		filtered := make([]api.KanbanCard, 0, len(cards))
		for _, card := range cards {
			if input.Namespace != "" && card.Namespace != input.Namespace {
				continue
			}
			if input.Stage != "" && string(card.Stage) != input.Stage {
				continue
			}
			filtered = append(filtered, card)
		}

		out := listKanbanCardsOutput{Cards: filtered, Total: len(filtered)}
		data, err := json.Marshal(out)
		if err != nil {
			return nil, listKanbanCardsOutput{}, fmt.Errorf("failed to marshal response: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, out, nil
	})

	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("kagent kanban MCP server listening at %s (kagent: %s)", *httpAddr, baseURL)
		return http.ListenAndServe(*httpAddr, handler)
	}

	t := &mcp.LoggingTransport{Transport: &mcp.StdioTransport{}, Writer: os.Stderr}
	return server.Run(context.Background(), t)
}
