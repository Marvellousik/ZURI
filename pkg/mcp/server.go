package mcp

import (
	"database/sql"
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewServerHandler(db *sql.DB) http.Handler {
	svc := NewMemoryService(db)

	getServer := func(r *http.Request) *mcpsdk.Server {
		server := mcpsdk.NewServer(&mcpsdk.Implementation{
			Name:    "zuri-brain",
			Version: "1.0.0",
		}, nil)

		mcpsdk.AddTool(server, &mcpsdk.Tool{
			Name:        "get_relevant_memory",
			Description: "Retrieve relevant engineering memory for a developer prompt and code context.",
		}, svc.HandleGetRelevantMemory)

		mcpsdk.AddTool(server, &mcpsdk.Tool{
			Name:        "resolve_memory",
			Description: "Confirm, reject, or edit a proposed probabilistic memory record.",
		}, svc.HandleResolveMemory)

		mcpsdk.AddTool(server, &mcpsdk.Tool{
			Name:        "resolve_knowledge_gap",
			Description: "Answer an open knowledge gap or acknowledge it as intentionally unknown per §13.4.",
		}, svc.HandleResolveKnowledgeGap)

		return server
	}

	return mcpsdk.NewStreamableHTTPHandler(getServer, nil)
}
