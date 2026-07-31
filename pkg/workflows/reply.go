package workflows

import (
	"context"
	"fmt"
	"zuri-daemon/pkg/github"
	"zuri-daemon/pkg/mcp"
)

// MemoryResolver defines the interface for resolving memory to allow mocking.
type MemoryResolver interface {
	HandleResolveMemory(ctx context.Context, req interface{}, input mcp.ResolveMemoryInput) (interface{}, mcp.ResolveMemoryOutput, error)
}

// ReplyProcessor orchestrates the workflow of parsing a GitHub reply and resolving memory.
type ReplyProcessor struct {
	resolver MemoryResolver
}

func NewReplyProcessor(resolver MemoryResolver) *ReplyProcessor {
	return &ReplyProcessor{resolver: resolver}
}

// Process processes a GitHub comment to conditionally update a proposed memory.
func (p *ReplyProcessor) Process(ctx context.Context, commentBody, memoryID, resolvedBy string, prMerged bool, appliesTo []string, sourceCtx string) error {
	parsed := github.ParseReply(commentBody)

	if parsed.Action == github.ActionNone {
		// Not a Zuri command, ignore it cleanly
		return nil
	}

	actionStr := string(parsed.Action)
	var editedContent string
	if parsed.Action == github.ActionEdit {
		editedContent = parsed.Content
	}

	input := mcp.ResolveMemoryInput{
		MemoryID:         memoryID,
		Action:           actionStr,
		EditedContent:    editedContent,
		ResolvedBy:       resolvedBy,
		PRMerged:         prMerged,
		AppliesToRepoIDs: appliesTo,
	}

	if sourceCtx != "" {
		input.SourceContext = &sourceCtx
	}

	// We pass nil for the mcpsdk.CallToolRequest since we are invoking the service internally.
	_, _, err := p.resolver.HandleResolveMemory(ctx, nil, input)
	if err != nil {
		return fmt.Errorf("failed to resolve memory via mcp service: %w", err)
	}

	return nil
}
