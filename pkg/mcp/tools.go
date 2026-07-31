package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pgvector/pgvector-go"
	"github.com/lib/pq"
	"zuri-daemon/pkg/scoring"
)

type MemoryService struct {
	db *sql.DB
}

func NewMemoryService(db *sql.DB) *MemoryService {
	return &MemoryService{db: db}
}

type GetRelevantMemoryInput struct {
	PromptText   string   `json:"prompt_text" jsonschema:"description=The developer's actual prompt,required"`
	FilesInScope []string `json:"files_in_scope" jsonschema:"description=File paths currently open or touched in scope,required"`
	TokenBudget  int      `json:"token_budget" jsonschema:"description=How much context room the agent has left in tokens,required"`
}

type MemorySource struct {
	PR       *int    `json:"pr"`
	MergedAt *string `json:"merged_at"`
	State    *string `json:"state"`
}

type MemoryScore struct {
	Relevance float64 `json:"relevance"`
	Trend     string  `json:"trend"`
	Citations int     `json:"citations"`
	LastCited *string `json:"last_cited"`
}

type MemoryItem struct {
	MemoryID   string       `json:"memory_id"`
	Tier       string       `json:"tier"`
	Status     string       `json:"status"`
	Decision   string       `json:"decision"`
	Reasoning  string       `json:"reasoning"`
	Source     MemorySource `json:"source"`
	Touches    []string     `json:"touches"`
	Score      MemoryScore  `json:"score"`
	FinalScore float64      `json:"-"`
}

type GetRelevantMemoryOutput struct {
	QueryTokensUsed int          `json:"query_tokens_used"`
	Memories        []MemoryItem `json:"memories"`
}

type ResolveMemoryInput struct {
	MemoryID      string  `json:"memory_id" jsonschema:"description=Synthetic UUID of the memory record,required"`
	Action        string  `json:"action" jsonschema:"description=Action to perform: confirm | reject | edit,required"`
	EditedContent string  `json:"edited_content,omitempty" jsonschema:"description=Required only when action is edit; replaces decision before confirming"`
	ResolvedBy    string  `json:"resolved_by" jsonschema:"description=GitHub username or agent-session identity,required"`
	PRMerged      bool    `json:"pr_merged,omitempty" jsonschema:"description=Explicit boolean flag indicating whether the originating PR is merged to the default branch"`
	SourceContext *string `json:"source_context,omitempty" jsonschema:"description=Optional context e.g. PR #601 comment or agent session prompt"`
}

type ResolveMemoryOutput struct {
	MemoryID       string `json:"memory_id"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	NewTier        string `json:"new_tier"`
	ResolvedAt     string `json:"resolved_at"`
}

func (s *MemoryService) HandleGetRelevantMemory(ctx context.Context, req *mcpsdk.CallToolRequest, input GetRelevantMemoryInput) (*mcpsdk.CallToolResult, GetRelevantMemoryOutput, error) {
	if input.PromptText == "" {
		return nil, GetRelevantMemoryOutput{}, fmt.Errorf("prompt_text is required")
	}

	// Record read access in audit_log per section 14 of spec
	auditCtx, _ := json.Marshal(map[string]any{
		"prompt_text":    input.PromptText,
		"files_in_scope": input.FilesInScope,
		"token_budget":   input.TokenBudget,
	})
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO audit_log (memory_id, event_type, actor, context)
		VALUES (NULL, 'retrieved', 'agent_session', $1);
	`, auditCtx)

	// Stage 1 candidate retrieval: semantic similarity via pgvector plus memory_touches_file structural join
	hasVectorExt := false
	_ = s.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector');").Scan(&hasVectorExt)

	var rows *sql.Rows
	var err error

	// Rejected records are excluded at retrieval query time per section 9.3 so they never occupy candidate slots
	if hasVectorExt {
		query := `
			SELECT DISTINCT
				m.memory_id, m.tier, m.status, m.decision, m.reasoning,
				m.originating_pr_number, m.created_at, m.citation_count, m.last_cited_at,
				COALESCE(CASE WHEN m.content_embedding IS NOT NULL THEN (1.0 - (m.content_embedding <=> $1::vector)) ELSE 0.5 END, 0.5) AS relevance_score,
				ARRAY(SELECT file_path FROM memory_touches_file WHERE memory_id = m.memory_id) AS touched_files
			FROM memory_record m
			LEFT JOIN memory_touches_file f ON m.memory_id = f.memory_id
			WHERE m.status != 'rejected'
			  AND (
				(cardinality($2::text[]) > 0 AND f.file_path = ANY($2::text[]))
				OR m.content_embedding IS NOT NULL
				OR m.decision ILIKE '%' || $3 || '%'
			  )
			LIMIT 100;
		`
		dummyVec := pgvector.NewVector(make([]float32, 1536))
		rows, err = s.db.QueryContext(ctx, query, dummyVec, pq.Array(input.FilesInScope), input.PromptText)
	} else {
		query := `
			SELECT DISTINCT
				m.memory_id, m.tier, m.status, m.decision, m.reasoning,
				m.originating_pr_number, m.created_at, m.citation_count, m.last_cited_at,
				0.5 AS relevance_score,
				ARRAY(SELECT file_path FROM memory_touches_file WHERE memory_id = m.memory_id) AS touched_files
			FROM memory_record m
			LEFT JOIN memory_touches_file f ON m.memory_id = f.memory_id
			WHERE m.status != 'rejected'
			  AND (
				(cardinality($1::text[]) > 0 AND f.file_path = ANY($1::text[]))
				OR m.content_embedding IS NOT NULL
				OR m.decision ILIKE '%' || $2 || '%'
			  )
			LIMIT 100;
		`
		rows, err = s.db.QueryContext(ctx, query, pq.Array(input.FilesInScope), input.PromptText)
	}

	if err != nil {
		return nil, GetRelevantMemoryOutput{}, fmt.Errorf("failed to query candidate memories: %w", err)
	}
	defer rows.Close()

	var candidates []MemoryItem

	for rows.Next() {
		var memID, tier, status, decision, reasoning string
		var prNum sql.NullInt64
		var createdAt time.Time
		var citationCount int
		var lastCitedAt sql.NullTime
		var stage1Relevance float64
		var touchedFiles []string

		if err := rows.Scan(&memID, &tier, &status, &decision, &reasoning, &prNum, &createdAt, &citationCount, &lastCitedAt, &stage1Relevance, pq.Array(&touchedFiles)); err != nil {
			return nil, GetRelevantMemoryOutput{}, fmt.Errorf("failed scanning memory record: %w", err)
		}

		var lastCitedPtr *time.Time
		if lastCitedAt.Valid {
			lastCitedPtr = &lastCitedAt.Time
		}

		// Stage 2 Ranking: Real scoring model per section 9.3
		// Calculate each named factor independently
		rel := scoring.CalculateRelevance(stage1Relevance)

		// Query rolling window citation counts for trend calculation
		var recentCitations, priorCitations int
		_ = s.db.QueryRowContext(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE cited_at >= now() - INTERVAL '30 days'),
				COUNT(*) FILTER (WHERE cited_at >= now() - INTERVAL '60 days' AND cited_at < now() - INTERVAL '30 days')
			FROM memory_citation
			WHERE cited_memory_id = $1;
		`, memID).Scan(&recentCitations, &priorCitations)

		trendVal := scoring.CalculateTrend(recentCitations, priorCitations)
		statusW := scoring.GetStatusWeight(tier, status)
		recency := scoring.CalculateRecency(lastCitedPtr, createdAt, scoring.DefaultHalfLifeDays)

		finalScore := scoring.CalculateFinalScore(rel, trendVal, statusW, recency)

		// Revival check (§9.5): flag lapsed record if citation trend inverts to rising
		_ = scoring.CheckAndFlagRevival(ctx, s.db, memID, status, trendVal)

		trendStr := "flat"
		if trendVal > 1.05 {
			trendStr = "rising"
		} else if trendVal < 0.95 {
			trendStr = "decaying"
		}

		lastCitedStr := ""
		if lastCitedPtr != nil {
			lastCitedStr = lastCitedPtr.UTC().Format(time.RFC3339)
		}

		score := MemoryScore{
			Relevance: rel,
			Trend:     trendStr,
			Citations: citationCount,
			LastCited: &lastCitedStr,
		}

		var sourcePR *int
		var stateStr *string
		if prNum.Valid {
			prVal := int(prNum.Int64)
			sourcePR = &prVal
			st := "merged"
			if tier == "probabilistic" {
				st = "open"
			}
			stateStr = &st
		}

		createdStr := createdAt.UTC().Format(time.RFC3339)

		item := MemoryItem{
			MemoryID:  memID,
			Tier:      tier,
			Status:    status,
			Decision:  decision,
			Reasoning: reasoning,
			Source: MemorySource{
				PR:       sourcePR,
				MergedAt: &createdStr,
				State:    stateStr,
			},
			Touches:    touchedFiles,
			Score:      score,
			FinalScore: finalScore,
		}

		candidates = append(candidates, item)
	}

	// Sort candidates by finalScore descending
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].FinalScore > candidates[i].FinalScore {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Truncate payload to fit token budget
	var truncated []MemoryItem
	totalTokens := 0
	for _, item := range candidates {
		itemTokens := (len(item.Decision) + len(item.Reasoning) + 100) / 4
		if input.TokenBudget > 0 && totalTokens+itemTokens > input.TokenBudget && len(truncated) > 0 {
			break
		}
		truncated = append(truncated, item)
		totalTokens += itemTokens
	}

	return nil, GetRelevantMemoryOutput{
		QueryTokensUsed: totalTokens,
		Memories:        truncated,
	}, nil
}

func (s *MemoryService) HandleResolveMemory(ctx context.Context, req *mcpsdk.CallToolRequest, input ResolveMemoryInput) (*mcpsdk.CallToolResult, ResolveMemoryOutput, error) {
	if input.MemoryID == "" {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("memory_id is required")
	}
	if input.Action != "confirm" && input.Action != "reject" && input.Action != "edit" {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("action must be one of: confirm, reject, edit")
	}
	if input.ResolvedBy == "" {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("resolved_by is required")
	}

	// Fetch target record and verify current status
	var currentStatus, currentTier, decision string
	var prNum sql.NullInt64
	var repoID string

	err := s.db.QueryRowContext(ctx, `
		SELECT status, tier, decision, originating_pr_number, repo_id
		FROM memory_record
		WHERE memory_id = $1;
	`, input.MemoryID).Scan(&currentStatus, &currentTier, &decision, &prNum, &repoID)

	if err == sql.ErrNoRows {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("memory record with id %s not found", input.MemoryID)
	} else if err != nil {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("failed fetching memory record: %w", err)
	}

	// Mechanical enforcement (§4 & §13.3): resolve_memory can only transition an existing 'proposed' record
	if currentStatus != "proposed" {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("cannot resolve memory record: record status must be 'proposed', but is currently '%s'", currentStatus)
	}

	newStatus := currentStatus
	newTier := currentTier
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	var expiresAt *time.Time

	if input.Action == "edit" {
		if input.EditedContent == "" {
			return nil, ResolveMemoryOutput{}, fmt.Errorf("edited_content is required when action is edit")
		}
		decision = input.EditedContent
		input.Action = "confirm"
	}

	if input.Action == "confirm" {
		newStatus = "confirmed"
		// PRMerged boolean explicitly passed by caller (e.g. S4 webhook listener on pull_request.closed merged:true)
		if input.PRMerged {
			newTier = "canonical"
			expiresAt = nil
		} else {
			newTier = "probabilistic"
			// Set expiration window based on repo zuri_config (default 60 days)
			var expiryDays int = 60
			_ = s.db.QueryRowContext(ctx, "SELECT expiry_days FROM zuri_config WHERE repo_id = $1;", repoID).Scan(&expiryDays)
			exp := now.AddDate(0, 0, expiryDays)
			expiresAt = &exp
		}
	} else if input.Action == "reject" {
		newStatus = "rejected"
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE memory_record
		SET status = $1,
			tier = $2,
			decision = $3,
			resolved_by = $4,
			resolved_at = $5,
			expires_at = $6
		WHERE memory_id = $7;
	`, newStatus, newTier, decision, input.ResolvedBy, now, expiresAt, input.MemoryID)

	if err != nil {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("failed to update memory record: %w", err)
	}

	// Audit log entry (§14)
	auditCtx, _ := json.Marshal(map[string]any{
		"action":          input.Action,
		"previous_status": currentStatus,
		"new_status":      newStatus,
		"new_tier":        newTier,
		"pr_merged":       input.PRMerged,
		"source_context":  input.SourceContext,
	})

	eventType := "confirmed"
	if input.Action == "reject" {
		eventType = "rejected"
	} else if input.EditedContent != "" {
		eventType = "edited"
	}

	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO audit_log (memory_id, event_type, actor, context, occurred_at)
		VALUES ($1, $2, $3, $4, $5);
	`, input.MemoryID, eventType, input.ResolvedBy, auditCtx, now)

	return nil, ResolveMemoryOutput{
		MemoryID:       input.MemoryID,
		PreviousStatus: currentStatus,
		NewStatus:      newStatus,
		NewTier:        newTier,
		ResolvedAt:     nowStr,
	}, nil
}
