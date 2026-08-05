package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/lib/pq"
	pgvector "github.com/pgvector/pgvector-go"

	"zuri-daemon/pkg/graph"
	"zuri-daemon/pkg/scoring"
)

type MemoryService struct {
	db *sql.DB
}

func NewMemoryService(db *sql.DB) *MemoryService {
	return &MemoryService{db: db}
}

type GetRelevantMemoryInput struct {
	PromptText   string   `json:"prompt_text" jsonschema_description:"Developer prompt or query describing intent"`
	FilesInScope []string `json:"files_in_scope,omitempty" jsonschema_description:"Array of relative or absolute file paths involved in the current code context"`
	TokenBudget  int      `json:"token_budget,omitempty" jsonschema_description:"Token budget for memory injection"`
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

type MemoryConfidence struct {
	ExtractionConfidence *float64 `json:"extraction_confidence"`
	EvidenceStrength     float64  `json:"evidence_strength"`
	Rationale            string   `json:"rationale"`
}

type MemoryItem struct {
	MemoryID   string           `json:"memory_id"`
	Tier       string           `json:"tier"`
	Status     string           `json:"status"`
	Decision   string           `json:"decision"`
	Reasoning  string           `json:"reasoning"`
	Source     MemorySource     `json:"source"`
	Touches    []string         `json:"touches"`
	Score      MemoryScore      `json:"score"`
	Confidence MemoryConfidence `json:"confidence"`
	FinalScore float64          `json:"-"`
}

type GetRelevantMemoryOutput struct {
	QueryTokensUsed int          `json:"query_tokens_used"`
	Memories        []MemoryItem `json:"memories"`
}

type ResolveMemoryInput struct {
	MemoryID         string   `json:"memory_id" jsonschema_description:"Synthetic UUID of the memory record"`
	Action           string   `json:"action" jsonschema_description:"Action to perform: confirm or reject or edit"`
	EditedContent    string   `json:"edited_content,omitempty" jsonschema_description:"Required only when action is edit"`
	ResolvedBy       string   `json:"resolved_by" jsonschema_description:"GitHub username or agent-session identity"`
	PRMerged         bool     `json:"pr_merged,omitempty" jsonschema_description:"Explicit boolean flag indicating whether originating PR is merged"`
	AppliesToRepoIDs []string `json:"applies_to_repo_ids,omitempty" jsonschema_description:"Optional array of additional repo UUIDs"`
	SourceContext    *string  `json:"source_context,omitempty" jsonschema_description:"Optional context"`
}

type ResolveMemoryOutput struct {
	MemoryID       string `json:"memory_id"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	NewTier        string `json:"new_tier"`
	ResolvedAt     string `json:"resolved_at"`
}

type ResolveKnowledgeGapInput struct {
	GapID         string `json:"gap_id" jsonschema_description:"Synthetic UUID of the knowledge gap"`
	Action        string `json:"action" jsonschema_description:"Action to perform: answer or acknowledge_unknown"`
	AnswerContent string `json:"answer_content,omitempty" jsonschema_description:"Required only when action is answer"`
	ResolvedBy    string `json:"resolved_by" jsonschema_description:"GitHub username or agent-session identity"`
}

type ResolveKnowledgeGapOutput struct {
	GapID      string `json:"gap_id"`
	NewStatus  string `json:"new_status"`
	MemoryID   string `json:"memory_id,omitempty"`
	ResolvedAt string `json:"resolved_at"`
}

func (s *MemoryService) resolveRepoScope(ctx context.Context, filesInScope []string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT repo_id, local_path FROM repo;")
	if err != nil {
		return nil, fmt.Errorf("failed to query repos for scope resolution: %w", err)
	}
	defer rows.Close()

	type repoInfo struct {
		id   string
		path string
	}
	var allRepos []repoInfo
	for rows.Next() {
		var r repoInfo
		if err := rows.Scan(&r.id, &r.path); err != nil {
			return nil, fmt.Errorf("failed scanning repo info: %w", err)
		}
		allRepos = append(allRepos, r)
	}

	if len(allRepos) == 0 {
		return nil, nil
	}

	implicatedMap := make(map[string]bool)

	if len(filesInScope) > 0 {
		for _, file := range filesInScope {
			normFile := filepath.ToSlash(filepath.Clean(file))
			for _, r := range allRepos {
				if r.path == "" {
					continue
				}
				normRepoPath := filepath.ToSlash(filepath.Clean(r.path))
				if strings.HasPrefix(normFile, normRepoPath) || strings.HasPrefix(normRepoPath, normFile) || strings.Contains(normFile, normRepoPath) {
					implicatedMap[r.id] = true
				}
			}
		}
	}

	if len(implicatedMap) == 0 {
		for _, r := range allRepos {
			implicatedMap[r.id] = true
		}
	}

	implicatedIDs := make([]string, 0, len(implicatedMap))
	for id := range implicatedMap {
		implicatedIDs = append(implicatedIDs, id)
	}

	return implicatedIDs, nil
}

func (s *MemoryService) HandleGetRelevantMemory(ctx context.Context, req *mcpsdk.CallToolRequest, input GetRelevantMemoryInput) (*mcpsdk.CallToolResult, GetRelevantMemoryOutput, error) {
	if input.PromptText == "" {
		return nil, GetRelevantMemoryOutput{}, fmt.Errorf("prompt_text is required")
	}

	auditCtx, _ := json.Marshal(map[string]any{
		"prompt_text":    input.PromptText,
		"files_in_scope": input.FilesInScope,
		"token_budget":   input.TokenBudget,
	})
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO audit_log (memory_id, event_type, actor, context)
		VALUES (NULL, 'retrieved', 'agent_session', $1);
	`, auditCtx)

	implicatedRepoIDs, err := s.resolveRepoScope(ctx, input.FilesInScope)
	if err != nil {
		return nil, GetRelevantMemoryOutput{}, err
	}
	if len(implicatedRepoIDs) == 0 {
		return nil, GetRelevantMemoryOutput{QueryTokensUsed: 0, Memories: []MemoryItem{}}, nil
	}

	hasVectorExt := false
	_ = s.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector');").Scan(&hasVectorExt)

	var rows *sql.Rows

	if hasVectorExt {
		query := `
			SELECT DISTINCT
				m.memory_id, m.tier, m.status, m.decision, m.reasoning,
				m.originating_pr_number, m.created_at, m.citation_count, m.last_cited_at,
				m.extraction_confidence, m.evidence_strength,
				COALESCE(CASE WHEN m.content_embedding IS NOT NULL THEN (1.0 - (m.content_embedding <=> $1::vector)) ELSE 0.5 END, 0.5) AS relevance_score,
				ARRAY(SELECT file_path FROM memory_touches_file WHERE memory_id = m.memory_id) AS touched_files
			FROM memory_record m
			LEFT JOIN memory_touches_file f ON m.memory_id = f.memory_id
			WHERE m.status != 'rejected'
			  AND (
				m.repo_id::text = ANY($2)
				OR EXISTS (
					SELECT 1 FROM memory_applies_to_repo mar
					WHERE mar.memory_id = m.memory_id AND mar.repo_id::text = ANY($2)
				)
			  )
			  AND (
				(cardinality($3::text[]) > 0 AND f.file_path = ANY($3::text[]))
				OR m.content_embedding IS NOT NULL
				OR m.decision ILIKE '%' || $4 || '%'
			  )
			LIMIT 100;
		`
		dummyVec := pgvector.NewVector(make([]float32, 1536))
		rows, err = s.db.QueryContext(ctx, query, dummyVec, pq.Array(implicatedRepoIDs), pq.Array(input.FilesInScope), input.PromptText)
	} else {
		query := `
			SELECT DISTINCT
				m.memory_id, m.tier, m.status, m.decision, m.reasoning,
				m.originating_pr_number, m.created_at, m.citation_count, m.last_cited_at,
				m.extraction_confidence, m.evidence_strength,
				0.5 AS relevance_score,
				ARRAY(SELECT file_path FROM memory_touches_file WHERE memory_id = m.memory_id) AS touched_files
			FROM memory_record m
			LEFT JOIN memory_touches_file f ON m.memory_id = f.memory_id
			WHERE m.status != 'rejected'
			  AND (
				m.repo_id::text = ANY($1)
				OR EXISTS (
					SELECT 1 FROM memory_applies_to_repo mar
					WHERE mar.memory_id = m.memory_id AND mar.repo_id::text = ANY($1)
				)
			  )
			  AND (
				(cardinality($2::text[]) > 0 AND f.file_path = ANY($2::text[]))
				OR m.content_embedding IS NOT NULL
				OR m.decision ILIKE '%' || $3 || '%'
			  )
			LIMIT 100;
		`
		rows, err = s.db.QueryContext(ctx, query, pq.Array(implicatedRepoIDs), pq.Array(input.FilesInScope), input.PromptText)
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
		var extConfidence sql.NullFloat64
		var evStrength float64
		var stage1Relevance float64
		var touchedFiles []string

		if err := rows.Scan(&memID, &tier, &status, &decision, &reasoning, &prNum, &createdAt, &citationCount, &lastCitedAt, &extConfidence, &evStrength, &stage1Relevance, pq.Array(&touchedFiles)); err != nil {
			return nil, GetRelevantMemoryOutput{}, fmt.Errorf("failed scanning memory record: %w", err)
		}

		var lastCitedPtr *time.Time
		if lastCitedAt.Valid {
			lastCitedPtr = &lastCitedAt.Time
		}

		rel := scoring.CalculateRelevance(stage1Relevance)

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

		if len(input.FilesInScope) > 0 && len(touchedFiles) > 0 {
			graphStore := graph.NewPostgresGraphStore(s.db)
			booster := graph.NewProximityBooster(graphStore, s.db)
			repoID := ""
			if len(implicatedRepoIDs) > 0 {
				repoID = implicatedRepoIDs[0]
			}
			boostedScore, _, _ := booster.ApplyProximityBoost(ctx, repoID, memID, touchedFiles[0], finalScore, input.FilesInScope)
			finalScore = boostedScore
		}

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

		var extConfPtr *float64
		if extConfidence.Valid {
			v := extConfidence.Float64
			extConfPtr = &v
		}

		confidence := MemoryConfidence{
			ExtractionConfidence: extConfPtr,
			EvidenceStrength:     evStrength,
			Rationale:            fmt.Sprintf("status: %s, citations: %d, tier: %s", status, citationCount, tier),
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
			Confidence: confidence,
			FinalScore: finalScore,
		}

		candidates = append(candidates, item)
	}

	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].FinalScore > candidates[i].FinalScore {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

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
		if input.PRMerged {
			newTier = "canonical"
			expiresAt = nil
		} else {
			newTier = "probabilistic"
			var expiryDays int = 60
			_ = s.db.QueryRowContext(ctx, "SELECT expiry_days FROM zuri_config WHERE repo_id = $1;", repoID).Scan(&expiryDays)
			exp := now.AddDate(0, 0, expiryDays)
			expiresAt = &exp
		}
	} else if input.Action == "reject" {
		newStatus = "rejected"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
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

	if input.Action == "confirm" && len(input.AppliesToRepoIDs) > 0 {
		for _, appRepoID := range input.AppliesToRepoIDs {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO memory_applies_to_repo (memory_id, repo_id)
				VALUES ($1, $2)
				ON CONFLICT (memory_id, repo_id) DO NOTHING;
			`, input.MemoryID, appRepoID)
			if err != nil {
				return nil, ResolveMemoryOutput{}, fmt.Errorf("failed to insert applicability row: %w", err)
			}
		}
	}

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

	_, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log (memory_id, event_type, actor, context, occurred_at)
		VALUES ($1, $2, $3, $4, $5);
	`, input.MemoryID, eventType, input.ResolvedBy, auditCtx, now)
	if err != nil {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("failed to insert audit log: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, ResolveMemoryOutput{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil, ResolveMemoryOutput{
		MemoryID:       input.MemoryID,
		PreviousStatus: currentStatus,
		NewStatus:      newStatus,
		NewTier:        newTier,
		ResolvedAt:     nowStr,
	}, nil
}

func (s *MemoryService) HandleResolveKnowledgeGap(ctx context.Context, req *mcpsdk.CallToolRequest, input ResolveKnowledgeGapInput) (*mcpsdk.CallToolResult, ResolveKnowledgeGapOutput, error) {
	if input.GapID == "" {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("gap_id is required")
	}
	if input.Action != "answer" && input.Action != "acknowledge_unknown" {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("action must be 'answer' or 'acknowledge_unknown'")
	}
	if input.ResolvedBy == "" {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("resolved_by is required")
	}

	var decisionKey, scope, status string
	err := s.db.QueryRowContext(ctx, `
		SELECT decision_key, scope, status
		FROM knowledge_gap
		WHERE gap_id = $1;
	`, input.GapID).Scan(&decisionKey, &scope, &status)

	if err == sql.ErrNoRows {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("knowledge gap with id %s not found", input.GapID)
	} else if err != nil {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("failed fetching knowledge gap: %w", err)
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	var newMemoryID string

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("failed starting transaction: %w", err)
	}
	defer tx.Rollback()

	if input.Action == "answer" {
		if input.AnswerContent == "" {
			return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("answer_content is required when action is 'answer'")
		}

		var repoID string
		_ = tx.QueryRowContext(ctx, "SELECT repo_id FROM repo WHERE github_repo_full_name = $1 OR repo_id::text = $1 LIMIT 1;", scope).Scan(&repoID)
		if repoID == "" {
			_ = tx.QueryRowContext(ctx, "SELECT repo_id FROM repo LIMIT 1;").Scan(&repoID)
		}

		err = tx.QueryRowContext(ctx, `
			INSERT INTO memory_record (
				repo_id, tier, status, source_type, decision_key, decision, reasoning,
				evidence_strength, created_by, resolved_by, resolved_at, created_at
			) VALUES (
				$1, 'canonical', 'confirmed', 'onboarding_survey', $2, $3, $4,
				0.65, $5, $5, $6, $6
			) RETURNING memory_id;
		`, repoID, decisionKey, input.AnswerContent, "Provided via knowledge gap answer", input.ResolvedBy, now).Scan(&newMemoryID)

		if err != nil {
			return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("failed creating memory record for gap answer: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE knowledge_gap
			SET status = 'answered', resolved_at = $1, resolved_by = $2
			WHERE gap_id = $3;
		`, now, input.ResolvedBy, input.GapID)

		if err != nil {
			return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("failed updating knowledge_gap status: %w", err)
		}

		auditCtx, _ := json.Marshal(map[string]any{
			"action":        "answer",
			"decision_key":  decisionKey,
			"new_memory_id": newMemoryID,
		})
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO audit_log (gap_id, memory_id, event_type, actor, context, occurred_at)
			VALUES ($1, $2, 'gap_answered', $3, $4, $5);
		`, input.GapID, newMemoryID, input.ResolvedBy, auditCtx, now)

	} else if input.Action == "acknowledge_unknown" {
		_, err = tx.ExecContext(ctx, `
			UPDATE knowledge_gap
			SET status = 'acknowledged_unknown', resolved_at = $1, resolved_by = $2
			WHERE gap_id = $3;
		`, now, input.ResolvedBy, input.GapID)

		if err != nil {
			return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("failed updating knowledge_gap status: %w", err)
		}

		auditCtx, _ := json.Marshal(map[string]any{
			"action":       "acknowledge_unknown",
			"decision_key": decisionKey,
		})
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO audit_log (gap_id, memory_id, event_type, actor, context, occurred_at)
			VALUES ($1, NULL, 'gap_acknowledged_unknown', $2, $3, $4);
		`, input.GapID, input.ResolvedBy, auditCtx, now)
	}

	if err := tx.Commit(); err != nil {
		return nil, ResolveKnowledgeGapOutput{}, fmt.Errorf("failed committing transaction: %w", err)
	}

	newStatus := "answered"
	if input.Action == "acknowledge_unknown" {
		newStatus = "acknowledged_unknown"
	}

	return nil, ResolveKnowledgeGapOutput{
		GapID:      input.GapID,
		NewStatus:  newStatus,
		MemoryID:   newMemoryID,
		ResolvedAt: nowStr,
	}, nil
}
