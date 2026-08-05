package db

import (
	"time"
)

// MemoryTier represents the tier enum ('canonical', 'probabilistic', 'working')
type MemoryTier string

const (
	TierCanonical     MemoryTier = "canonical"
	TierProbabilistic MemoryTier = "probabilistic"
	TierWorking       MemoryTier = "working"
)

// MemoryStatus represents the status enum ('proposed', 'confirmed', 'rejected', 'lapsed')
type MemoryStatus string

const (
	StatusProposed  MemoryStatus = "proposed"
	StatusConfirmed MemoryStatus = "confirmed"
	StatusRejected  MemoryStatus = "rejected"
	StatusLapsed    MemoryStatus = "lapsed"
)

// MemorySourceType represents the source_type enum ('pr_merge', 'onboarding_survey', 'agent_session')
type MemorySourceType string

const (
	SourcePRMerge          MemorySourceType = "pr_merge"
	SourceOnboardingSurvey MemorySourceType = "onboarding_survey"
	SourceAgentSession     MemorySourceType = "agent_session"
)

// KnowledgeGapType represents the gap_type enum for knowledge gaps
type KnowledgeGapType string

const (
	KnowledgeGapConflictingConventions KnowledgeGapType = "conflicting_conventions"
	KnowledgeGapInsufficientEvidence     KnowledgeGapType = "insufficient_evidence"
	KnowledgeGapUnownedDecision         KnowledgeGapType = "unowned_decision"
	KnowledgeGapStaleUnreinforced       KnowledgeGapType = "stale_unreinforced"
)

// KnowledgeGapStatus represents the status enum for knowledge gaps
type KnowledgeGapStatus string

const (
	KnowledgeGapStatusOpen                KnowledgeGapStatus = "open"
	KnowledgeGapStatusSurfaced            KnowledgeGapStatus = "surfaced"
	KnowledgeGapStatusAnswered            KnowledgeGapStatus = "answered"
	KnowledgeGapStatusAcknowledgedUnknown KnowledgeGapStatus = "acknowledged_unknown"
	KnowledgeGapStatusStale               KnowledgeGapStatus = "stale"
)

// AuditEventType represents the event_type enum for audit logs
type AuditEventType string

const (
	AuditEventRetrieved               AuditEventType = "retrieved"
	AuditEventConfirmed               AuditEventType = "confirmed"
	AuditEventRejected                AuditEventType = "rejected"
	AuditEventEdited                  AuditEventType = "edited"
	AuditEventLapsed                  AuditEventType = "lapsed"
	AuditEventRevivalFlagged          AuditEventType = "revival_flagged"
	AuditEventGapDetected             AuditEventType = "gap_detected"
	AuditEventGapSurfaced             AuditEventType = "gap_surfaced"
	AuditEventGapAnswered             AuditEventType = "gap_answered"
	AuditEventGapAcknowledgedUnknown  AuditEventType = "gap_acknowledged_unknown"
)

// Repo maps to the `repo` table
type Repo struct {
	RepoID               string    `json:"repo_id"`
	GithubInstallationID int64     `json:"github_installation_id"`
	GithubRepoFullName   string    `json:"github_repo_full_name"`
	DefaultBranch        string    `json:"default_branch"`
	LocalPath            string    `json:"local_path"`
	CreatedAt            time.Time `json:"created_at"`
}

// ZuriConfig maps to the `zuri_config` table
type ZuriConfig struct {
	RepoID                   string   `json:"repo_id"`
	ApproverUsernames        []string `json:"approver_usernames"`
	ExpiryDays               int      `json:"expiry_days"`
	ReminderCadenceDays      int      `json:"reminder_cadence_days"`
	AdditionalNotifyChannels string   `json:"additional_notify_channels"`
}

// MemoryRecord maps to the `memory_record` table (Spec v1.1 & RFC §7.4)
type MemoryRecord struct {
	MemoryID                       string           `json:"memory_id"`
	RepoID                         string           `json:"repo_id"`
	Tier                           MemoryTier       `json:"tier"`
	Status                         MemoryStatus     `json:"status"`
	SourceType                     MemorySourceType `json:"source_type"`
	DecisionKey                    *string          `json:"decision_key,omitempty"`
	Concern                        *string          `json:"concern,omitempty"`
	DecisionType                   *string          `json:"decision_type,omitempty"`
	Boundary                       *string          `json:"boundary,omitempty"`
	Decision                       string           `json:"decision"`
	Reasoning                      string           `json:"reasoning"`
	ContentEmbedding               []float32        `json:"content_embedding,omitempty"`
	OriginatingCommit              *string          `json:"originating_commit,omitempty"`
	OriginatingPRNumber            *int             `json:"originating_pr_number,omitempty"`
	ModelID                        *string          `json:"model_id,omitempty"`
	ExtractionConfidenceRaw        *float64         `json:"extraction_confidence_raw,omitempty"`
	ExtractionConfidence           *float64         `json:"extraction_confidence,omitempty"`
	EvidenceStrength               float64          `json:"evidence_strength"`
	EvidenceStrengthFormulaVersion int              `json:"evidence_strength_formula_version"`
	CreatedBy                      string           `json:"created_by"`
	ResolvedBy                     *string          `json:"resolved_by,omitempty"`
	BranchLabel                    *string          `json:"branch_label,omitempty"`
	DecisionTitle                  *string          `json:"decision_title,omitempty"`
	CreatedAt                      time.Time        `json:"created_at"`
	ResolvedAt                     *time.Time       `json:"resolved_at,omitempty"`
	ExpiresAt                      *time.Time       `json:"expires_at,omitempty"`
	CitationCount                  int              `json:"citation_count"`
	LastCitedAt                    *time.Time       `json:"last_cited_at,omitempty"`
	LastRemindedAt                 *time.Time       `json:"last_reminded_at,omitempty"`
}

// ModelCalibration maps to the `model_calibration` table (RFC §7.4)
type ModelCalibration struct {
	ModelID          string    `json:"model_id"`
	Concern          string    `json:"concern"`
	CalibrationCurve string    `json:"calibration_curve"` // JSON string representation
	SampleSize       int       `json:"sample_size"`
	LastUpdatedAt    time.Time `json:"last_updated_at"`
}

// KnowledgeGap maps to the `knowledge_gap` table (§7.2, §10.7)
type KnowledgeGap struct {
	GapID               string             `json:"gap_id"`
	DecisionKey         string             `json:"decision_key"`
	Scope               string             `json:"scope"`
	GapType             KnowledgeGapType   `json:"gap_type"`
	CandidateHypotheses string             `json:"candidate_hypotheses"` // JSON array
	AffectedMemoryIDs   []string           `json:"affected_memory_ids"`
	Status              KnowledgeGapStatus `json:"status"`
	RoutedTo            []string           `json:"routed_to"`
	DetectedAt          time.Time          `json:"detected_at"`
	LastSurfacedAt      *time.Time         `json:"last_surfaced_at,omitempty"`
	ResolvedAt          *time.Time         `json:"resolved_at,omitempty"`
	ResolvedBy          *string            `json:"resolved_by,omitempty"`
}

// MemoryTouchesFile maps to the `memory_touches_file` join table
type MemoryTouchesFile struct {
	MemoryID string `json:"memory_id"`
	FilePath string `json:"file_path"`
}

// MemoryAppliesToRepo maps to the `memory_applies_to_repo` join table
type MemoryAppliesToRepo struct {
	MemoryID string `json:"memory_id"`
	RepoID   string `json:"repo_id"`
}

// MemoryCitation maps to the `memory_citation` table
type MemoryCitation struct {
	CitationID    string    `json:"citation_id"`
	CitingPRNumber int       `json:"citing_pr_number"`
	CitedMemoryID string    `json:"cited_memory_id"`
	CitedAt       time.Time `json:"cited_at"`
}

// AuditLog maps to the `audit_log` table
type AuditLog struct {
	LogID      string         `json:"log_id"`
	MemoryID   *string        `json:"memory_id,omitempty"`
	GapID      *string        `json:"gap_id,omitempty"`
	EventType  AuditEventType `json:"event_type"`
	Actor      string         `json:"actor"`
	Context    string         `json:"context"`
	OccurredAt time.Time      `json:"occurred_at"`
}
