# Zuri — Engineering Memory System
## Full Working Spec (v1.1 — Build-Ready)

This document supersedes `zuri-spec-v1.md` (v1.0). It integrates the evidence-first philosophy (formerly drafted as a standalone "Core Principles" section) directly into the existing structure, and adds the schema/pipeline elements needed to make that philosophy mechanically real rather than aspirational. Every section number from v1.0 is preserved unchanged — see §1.5 for why. New or materially changed subsections are marked **[NEW]** or **[AMENDED]** in their heading; unmarked sections are unchanged from v1.0.

---

## 1. Purpose & Core Principles

A second brain for a company's engineering organization — good enough that someone who **cannot code** can still direct production-quality changes to the codebase, because the AI coding agent they use is backed by a brain that knows the architecture, the reasoning, the history, and the constraints better than most individual senior developers do.

This is not a search tool for humans. It is an **MCP server** that sits between a coding agent (Claude Code, Gemini CLI, Antigravity, Cursor, etc.) and the company's accumulated engineering memory. On every prompt, the agent queries the brain first, receives relevant context, and only then generates code.

### 1.1 Core Principle: Evidence Over Certainty **[NEW]**

Zuri is an evidence-first engineering memory system, not a certainty-first system. Engineering knowledge is often incomplete, contradictory, or lost over time. Zuri does not attempt to manufacture certainty where none exists. Instead of claiming absolute truth, Zuri measures confidence based on available evidence and makes both the confidence and its supporting evidence visible.

The objective is not to maximize AI confidence. The objective is to maximize justified confidence.

Every memory in Zuri must answer two questions:

1. Why does Zuri believe this?
2. How strongly should an engineer trust it?

Every memory exposes: confidence, provenance, supporting evidence, lifecycle state, why the system believes it, and what evidence could increase or decrease confidence. A frontier model is treated as a reasoning engine, not as the source of truth. Truth is established through evidence.

### 1.2 Architectural Invariant: Models Are Replaceable **[NEW]**

No component of Zuri assumes any particular LLM is correct. The extraction pipeline (§10.2) is model-agnostic: models generate hypotheses about engineering decisions; Zuri evaluates, stores, and evolves those hypotheses using evidence, citations, historical usage, human review, and subsequent engineering activity.

Model intelligence is intentionally decoupled from system trustworthiness. A better frontier model improves candidate generation — better hypotheses, faster, with fewer misreadings of source artifacts. It does not, by itself, make a memory more trustworthy. Changing the underlying model must never require redesigning the memory lifecycle.

**Concrete rule:** no table, schema, or scoring formula in this spec may key its logic on which model produced a hypothesis. A model identifier is recorded as provenance metadata only (§7.2, `memory_record.model_id`). It may be used to *calibrate* how that model's stated confidence is interpreted (§1.4, §7.2 `model_calibration`), but never as a *source* of trust.

### 1.3 Confidence Model **[NEW]**

Every memory carries two independent assessments, stored separately so neither can be inferred from or collapse into the other.

**Extraction confidence** — how confident the reasoning engine was that it interpreted the available artifacts correctly. Derived from model reasoning quality, corrected against a per-model calibration curve (§1.4) rather than taken at face value. Populated only for memory produced by the extraction pipeline (§10.2); null for onboarding-survey-derived memory (§10.5), where there was no artifact to interpret, only a direct assertion.

**Evidence strength** — how strongly the available engineering artifacts support this memory, regardless of model confidence. Derived from objective signals: merged PRs, review approvals, architectural discussions, historical citations, later reinforcing changes, conflicting evidence, recency, ownership signals. This spec already computes most of these inputs (§9.3's `trend`, `citation_count`, `status_weight`) — §9.3 is amended to name this computation explicitly as evidence strength rather than leaving it as an unnamed input to ranking.

These scores are independent by design. A memory with weak evidence stays weak even at high extraction confidence. Strong evidence may still warrant human review if extraction confidence is low. Confidence is dynamic: it strengthens through citation and reuse, and weakens through contradiction, supersession, or revocation (§6, §9.5 already implement this dynamism for evidence strength; §1.4 extends it to extraction confidence via calibration).

**Ranking vs. trust — not the same question.** §9.3's scoring formula governs retrieval ordering: what's most relevant to surface right now. It answers an operational question, not a trust question. Extraction confidence and evidence strength are exposed to the calling agent and to humans via the GUI (§13.2, §12) as a transparency layer *on top of* whatever the ranking formula selected. Extraction confidence deliberately does **not** enter the ranking formula (§9.3) — a confirmed memory should not rank lower because the original model guess was shaky, since confirmation is stronger evidence than the guess it replaced. It does feed review-priority triage during backfill (§10.6) and gap surfacing (§10.7), where the question is genuinely "how much should a human trust this enough to skip reviewing it."

### 1.4 Model Calibration **[NEW]**

Because models vary in how well-calibrated their stated confidence is, and that calibration can drift when a model is swapped or upgraded, raw extraction confidence is never used directly. Zuri maintains a calibration curve per `(model_id, concern)` pair (§7.4): given this model's stated confidence on a decision within this engineering concern, what has the historical relationship been to the memory's eventual evidence-backed outcome (confirmed, edited, rejected)? Extraction confidence is corrected against this curve before being written to `memory_record` or surfaced to a caller.

This is the mechanism that makes §1.2's invariant true in the meaningful sense, not just the trivial sense that the LLM API endpoint is swappable: the system actually gets better or worse at trusting a given model's stated confidence, based on outcomes, without requiring any change to the memory lifecycle itself. See §7.2 and §7.4 for the `model_calibration` table and §10.2 for where the raw confidence is produced.

### 1.5 Knowledge Recovery **[NEW]**

The purpose of ingestion is not merely to extract engineering memory. It is to recover organizational knowledge. Zuri distinguishes three states for any given decision area: **known** (sufficient evidence exists — a canonical memory), **inferred** (a hypothesis with some support, not yet canonical, carrying visible confidence rather than being presented as settled), and **unknown** (evidence is insufficient or conflicting, and no reliable hypothesis exists).

Unknown knowledge is a first-class outcome, not a failure. When sufficient evidence doesn't exist, Zuri surfaces the gap instead of fabricating certainty. Onboarding, reviewer confirmation, and ongoing engineering activity exist to progressively close those gaps. **The objective is not to eliminate uncertainty. The objective is to continuously reduce uncertainty while preserving provenance.** See §7.2 (`knowledge_gap` table) and §10.7 (detection and surfacing mechanics) for the concrete implementation.

**A note on numbering:** this content was originally scoped as a standalone "Section 2: Core Principles." It's placed here instead because this spec's existing Section 2 is "Core Loop," and every later section cross-references others by number all the way through the build order in §16 — renumbering would touch every section for no functional benefit. If a standalone top-level section is still wanted for readability, it can be extracted from §1.1–1.5 later as a pure documentation reorganization; nothing below depends on it living in one place versus the other.

---

## 2. Core Loop

```
Developer prompt
      ↓
Coding agent (Claude Code, Gemini CLI, etc.)
      ↓
MCP client call → Zuri MCP server (get_relevant_memory)
      ↓
Brain returns relevant memory (decisions, conventions, past bugs, ownership, constraints)
      ↓
Agent generates code with that context injected
      ↓
Code goes through PR review
      ↓
Merge event promotes/updates memory (GitHub webhook → resolve_memory / ingestion pipeline)
```

The brain is not a chatbot you ask questions of. It's a standing local service the agent consults **automatically, every time**, scoped to whatever files/symbols/prompt are in play.

---

## 3. Memory Tiers

### 3.1 Canonical Memory
- Verified truth — established the same way code is: **by merging to main**.
- A PR merge is a memory-write event, not just a code-write event.
- High trust. Persists until explicitly superseded by a later canonical write.
- `status` is implicitly `confirmed` for all canonical records — there is no other valid status for this tier.
- Per §1.3, canonical status is itself the strongest evidence-strength signal, but it is not the only one tracked (§9.3) — a canonical memory can still be cited, reinforced, or contradicted after the fact, and its evidence strength continues to evolve.

### 3.2 Probabilistic Memory
- Decisions in flight — proposed, discussed, not yet implemented.
- Example: "migrating from SQLite to Postgres for pgvector support" — true-ish, but not final.
- Status lifecycle: `proposed → confirmed` (promotes to canonical, only if the originating PR is merged to main) or `proposed → lapsed` (see §6).
- Requires a named approver/gate (configurable — see §5).
- Has an expiry window. On expiry, memory does **not** get deleted — see §6.

### 3.3 Working Memory (per-developer, per-branch)
- A developer's in-progress architectural reasoning on a branch that hasn't merged yet.
- Personal scratchpad — follows the dev's current session/branch.
- Feeds the agent so it stays consistent with decisions made earlier in the same branch (e.g., "I architected this REST pattern this way three commits ago").
- Never leaks into canonical or probabilistic memory until it goes through a PR.
- At merge: promoted to canonical (if merged) or discarded (if abandoned) — same as code. "Discarded" here means excluded from active retrieval, not row-deleted (see §6 principle applied uniformly).

---

## 4. Conflict Resolution & Write Gating

**No direct writes to canonical memory — ever. No raw INSERT/UPDATE path exists in the codebase for canonical records.** Memory review is gated by code review or explicit founder confirmation, exactly like code:

- On-site developers can't merge to main directly → they also can't write to canonical memory directly.
- A PR that changes canonical memory carries the proposed memory-write alongside it.
- Same reviewers, same approval flow, same moment of landing. Memory is never approved faster or slower than the code describing it.
- No parallel path where memory could bypass the gate that code has to go through.
- **Mechanical enforcement**: the only write paths into `memory_record` are:
  (a) the GitHub ingestion pipeline (§10), for PR-derived memory;
  (b) the onboarding ingestion flow (§10.5), which may create canonical records only after explicit founder confirmation during Create Brain; and
  (c) the `resolve_memory` MCP tool (§13.3), which only transitions existing `proposed` records and cannot create a new canonical record.

  There is no admin panel, direct database endpoint, or CLI command that bypasses one of these review-gated flows. Use fixtures/migrations for testing and seeding.

  This preserves the architectural principle:
  - **PR-derived truth** → gated by code review.
  - **Greenfield truth** → gated by founder confirmation.
  - **Runtime confirmation** → only via state transition, never direct insertion.

  So the invariant becomes **"every canonical record must pass through an explicit review gate,"** rather than "every canonical record originates from a GitHub webhook." This gate is also, per §1.1, the mechanism by which extraction confidence becomes irrelevant to final trust — nothing is canonical because a model was sure of it; it's canonical because a human confirmed it through this gate.

---

## 5. Confidence Threshold

- **Mechanism is fixed, values are configurable.** Every org gets: status (`proposed`/`confirmed`/`lapsed`) + named approver + expiry window.
- Default out of the box (small teams): one named approver, 30–90 day expiry.
- Configurable per org: larger orgs may want multiple approvers / formal RFC sign-off before a probabilistic decision is trusted enough to shape generated code. A small startup may want just one tech lead's approval.
- The system does not hardcode organizational hierarchy — only the state machine. Approver identity is stored as a config value (GitHub username/team), not derived from any assumed org chart.
- **Implementation note**: this config lives in a per-repo `zuri_config` table (see §7.2), editable from the Electron GUI Settings pane for that project — not a flat file, so it survives GUI-driven edits and displays in the "container inspector" UI described in §12.

*Note: "Confidence Threshold" here refers to the approval-gate mechanism, distinct from the extraction-confidence/evidence-strength scores in §1.3. This section's threshold governs when a human decision becomes trusted; §1.3's scores describe how trusted an AI-inferred hypothesis is before and regardless of that human decision.*

---

## 6. Lapsed Memory (Expiry Handling)

When probabilistic memory's window expires unresolved:

- It is **never silently deleted.**
- It is marked `lapsed` — downgraded so the agent stops treating it as live guidance, but the record persists permanently in `memory_record`.
- The original proposer (and configured stakeholders) get notified: *"This decision was never confirmed — still relevant, abandoned, or needs a new deadline?"*
- Rationale: since AI systems (not human time) maintain the memory, there's no cost reason to drop information. Even "we considered this and never decided" is institutional memory worth keeping.
- **Retrieval ranking must reliably deprioritize lapsed/abandoned entries** — this is handled via the `status_weight` multiplier in the scoring model (§9.3), not via deletion or filtering at query time (filtering would make revival detection in §9.5 impossible).

---

## 7. Keying Principle & Schema

### 7.1 Keying Principle

Branch names, `main`/`master`, and other human-readable labels are **mutable and reusable** — they get renamed, deleted, force-pushed over, or reused for unrelated work later. Git itself never treats branch names as identity; a branch is just a movable pointer to a commit hash, which is the actual stable, content-addressed identifier. Zuri's memory keying follows the same separation:

- **Identity = a synthetic, generated ID (UUID)**, created once per memory record and never reused.
- **Labels (branch name, decision title, etc.) = mutable attributes** that point *at* the identity, used for human-friendly lookup — not the key itself.

This avoids orphaned memory when branches are deleted, key collisions when branch names get reused, and ambiguity when multiple developers share a long-running branch.

### 7.2 Full Schema (Postgres + pgvector — see §11 for why no separate graph DB) **[AMENDED]**

**`repo`**
| Field | Type | Notes |
|---|---|---|
| `repo_id` | UUID (PK) | synthetic identity |
| `github_installation_id` | bigint | from GitHub App install, not repo name/URL |
| `github_repo_full_name` | text | mutable label, e.g. `org/repo` — for display only |
| `default_branch` | text | e.g. `main` — mutable label |
| `local_path` | text | absolute local filesystem path where this repo is cloned on the machine running the daemon, registered when the repo is connected via the GUI (§12). Used to resolve which repo(s) a given file in `files_in_scope` belongs to, see §9.6. |
| `created_at` | timestamptz | |

**`memory_applies_to_repo`** (join table for cross-repo integration decisions, see §9.6)
| Field | Type | Notes |
|---|---|---|
| `memory_id` | FK → `memory_record` | |
| `repo_id` | FK → `repo` | a repo, in addition to the memory's originating repo, that this decision is also explicitly relevant to |

**`zuri_config`** (per-repo, editable via GUI — see §5) **[AMENDED]**
| Field | Type | Notes |
|---|---|---|
| `repo_id` | FK → `repo` | |
| `approver_usernames` | text[] | GitHub usernames authorized to confirm probabilistic memory |
| `expiry_days` | int | default 60 |
| `reminder_cadence_days` | int | default 7 — see §10.4 floor rule |
| `additional_notify_channels` | jsonb | optional Slack webhook etc. — post-v1 stub field, unused in v1 |
| `backfill_budget` | jsonb | `{type: 'tokens' \| 'pr_count', limit: int}` — see §10.6 |
| `gap_digest_cadence_days` | int | default 7 — see §10.7; distinct from `reminder_cadence_days`, which governs merged-but-unconfirmed decisions, not knowledge gaps |

**`memory_record`** (shared table across all tiers) **[AMENDED]**
| Field | Type | Notes |
|---|---|---|
| `memory_id` | UUID (PK) | synthetic, immutable identity |
| `repo_id` | FK → `repo` | not a literal repo name/URL |
| `tier` | enum | `canonical` \| `probabilistic` \| `working` |
| `status` | enum | `proposed` \| `confirmed` \| `rejected` \| `lapsed` (canonical is implicitly `confirmed`) |
| `source_type` | enum | `pr_merge` \| `onboarding_survey` \| `agent_session`, defaults to `pr_merge`. Distinguishes how a record entered the system; see §10.5 for the onboarding path. |
| `decision_key` | text, nullable | **[NEW]** normalized key derived from classification fields: `boundary:<boundary>/concern:<concern>/decision_type:<decision_type>` per §7.4. Used for conflict detection (§1.3), knowledge-gap clustering (§10.7), and multi-branch decision grouping. |
| `concern` | text, nullable | **[NEW]** normalized engineering area, e.g. `reliability`, `security`, `data`, `architecture` per §7.4 |
| `decision_type` | text, nullable | **[NEW]** specific decision kind within concern, e.g. `retry-policy`, `schema-design` per §7.4 |
| `boundary` | text, nullable | **[NEW]** concrete system boundary affected, e.g. `payments`, `checkout` per §7.4 |
| `decision` | text | the inferred/confirmed decision, one to two sentences |
| `reasoning` | text | supporting context — why, not just what |
| `content_embedding` | vector(1536) | pgvector column, generated from `decision` + `reasoning` at write time |
| `originating_commit` | text | commit hash at time of creation — stable, unlike branch name |
| `originating_pr_number` | int | nullable — null for pure working memory not yet in a PR |
| `model_id` | text, nullable | **[NEW]** identifier of the model that produced this extraction (e.g. `claude-sonnet-4-6`), provenance metadata only per §1.2 — never used as a trust signal, only as a key into `model_calibration`. Null for `source_type='onboarding_survey'`. |
| `extraction_confidence_raw` | float, nullable | **[NEW]** the model's self-reported confidence at extraction time, before calibration. Null for onboarding-derived memory. |
| `extraction_confidence` | float, nullable | **[NEW]** `extraction_confidence_raw` corrected against `model_calibration` for `(model_id, concern)` per §7.4. This is the field surfaced to callers (§13.2), never the raw value. |
| `evidence_strength` | float | **[NEW]** materialized view over `citation_count`, `status`, `tier`, `last_cited_at`, and any `knowledge_gap` resolution that seeded this memory (§10.7) — this is the same computation §9.3 already performs as an input to ranking, now also stored and exposed as its own named field rather than only existing inline inside the ranking formula. Recomputed whenever `memory_citation` gains a row or `status` changes. |
| `evidence_strength_formula_version` | int | **[NEW]** so the formula can be revised later and every historical record's `evidence_strength` recomputed without touching extraction or the model layer, per §1.2. |
| `created_by` | text | GitHub username or agent-session identity |
| `resolved_by` | text | nullable — who confirmed/rejected/edited |
| `branch_label` | text | mutable, human-readable — nullable for canonical (branch may be deleted) |
| `decision_title` | text | mutable label for probabilistic memory — a decision can span multiple branches/PRs over time |
| `created_at` | timestamptz | |
| `resolved_at` | timestamptz | nullable |
| `expires_at` | timestamptz | nullable — only set for `probabilistic` + `proposed` |
| `citation_count` | int | denormalized counter, updated by trigger on `memory_citation` insert |
| `last_cited_at` | timestamptz | denormalized, same trigger |

**`model_calibration`** **[AMENDED / NEW]**
| Field | Type | Notes |
|---|---|---|
| `model_id` | text | model provenance identifier |
| `concern` | text | normalized decision concern being calibrated (§7.4) |
| `calibration_curve` | jsonb | mapping from raw confidence to corrected confidence |
| `sample_size` | int | resolved outcomes used for calibration |
| `last_updated_at` | timestamptz | |

**`knowledge_gap`** **[NEW]**
| Field | Type | Notes |
|---|---|---|
| `gap_id` | UUID (PK) | |
| `decision_key` | text | constructed as `boundary:<boundary>/concern:<concern>/decision_type:<decision_type>` per §7.4 |
| `scope` | text | e.g. repo or service the gap concerns |
| `gap_type` | enum | `conflicting_conventions` \| `insufficient_evidence` \| `unowned_decision` \| `stale_unreinforced` |
| `candidate_hypotheses` | jsonb | array of weak memory drafts that existed but couldn't be promoted, if any |
| `affected_memory_ids` | UUID[] | memories this gap relates to or blocks |
| `status` | enum | `open` \| `surfaced` \| `answered` \| `acknowledged_unknown` \| `stale` |
| `routed_to` | text[] | GitHub usernames/teams derived from ownership signals, see §10.7 |
| `detected_at` | timestamptz | |
| `last_surfaced_at` | timestamptz, nullable | |
| `resolved_at` | timestamptz, nullable | |
| `resolved_by` | text, nullable | |

**`memory_touches_file`** (join table — this is what replaces a graph DB, see §11)
| Field | Type | Notes |
|---|---|---|
| `memory_id` | FK → `memory_record` | |
| `file_path` | text | relative path within repo |

Indexed on `file_path` for fast "what memories touch this file" lookups. Recursive CTEs over this table + `memory_citation` give 1–2 hop traversal ("what else touches files this memory touches") without a dedicated graph engine.

**`memory_citation`** (edge table for the citation/influence graph — §9.3 PageRank-style scoring)
| Field | Type | Notes |
|---|---|---|
| `citation_id` | UUID (PK) | |
| `citing_pr_number` | int | the PR/commit doing the citing |
| `cited_memory_id` | FK → `memory_record` | the memory being referenced |
| `cited_at` | timestamptz | |

**`audit_log`** (append-only — see §14) **[AMENDED]**
| Field | Type | Notes |
|---|---|---|
| `log_id` | UUID (PK) | |
| `memory_id` | FK → `memory_record`, nullable | null for pure read events with no single memory match |
| `gap_id` | FK → `knowledge_gap`, nullable | **[NEW]** null for events not related to a knowledge gap |
| `event_type` | enum | `retrieved` \| `confirmed` \| `rejected` \| `edited` \| `lapsed` \| `revival_flagged` \| `gap_detected` \| `gap_surfaced` \| `gap_answered` \| `gap_acknowledged_unknown` **[NEW values added]** |
| `actor` | text | GitHub username, agent-session ID, or `system` |
| `context` | jsonb | free-form — e.g. prompt text hash, PR number, source_context string |
| `occurred_at` | timestamptz | |

### 7.3 Keying by Tier
- **Canonical memory** → keyed by `memory_id`, no meaningful branch label (branch is likely deleted by the time it's canonical).
- **Working memory** → keyed by `memory_id`, indexed via `created_by` + `branch_label` for lookup, but never addressed *by* that label — the label can change/disappear without orphaning the record.
- **Probabilistic memory** → keyed by `memory_id`, indexed via `decision_title`, since a single decision can span multiple branches/PRs over time rather than living on one.

### 7.4 Decision Classification Taxonomy RFC [NEW]

#### Purpose

Zuri requires stable classification for decisions so that conflict detection, knowledge-gap clustering, retrieval, and future graph integration operate on consistent identifiers. The classification system must avoid reproducing the same ambiguity it is designed to remove.

The previous concept of a `decision_domain` hierarchy is rejected. A freeform hierarchy introduces a second competing taxonomy alongside `decision_key`, where the same concept can exist simultaneously as a parent node, a subject label, or a decision type. This creates classification drift rather than preventing it.

#### Decision Classification Model

Decision classification is represented by bounded fields:

* `concern`
* `decision_type`
* `boundary`

These fields replace the need for a separate `decision_domain` hierarchy.

##### `concern`

`concern` represents the engineering area affected by the decision.

Examples:
* reliability
* security
* performance
* data
* architecture
* deployment
* observability

`concern` is a controlled enum:
* seeded with a default set maintained by Zuri,
* extensible per organization,
* changes governed through the same explicit configuration ownership model as `zuri_config.approver_usernames`.

Organizations may add new concerns when their architecture requires them, but new values become part of the organization's controlled vocabulary rather than arbitrary extraction output.

##### `decision_type`

`decision_type` represents the specific kind of engineering decision being made within a concern.

Examples:

For `concern=reliability`:
* retry-policy
* timeout-policy
* circuit-breaker
* fallback-strategy

For `concern=data`:
* storage-selection
* schema-design
* migration-strategy

`decision_type` is also a bounded, org-extensible enum.

It is not a hierarchy node. A retry policy is not a child of reliability in a tree. Instead:

```
concern = reliability
decision_type = retry-policy
```

This preserves classification stability without requiring a second graph structure.

#### Boundary Resolution

`boundary` identifies the concrete system boundary affected by a decision.

It is not an enum and does not contain abstract categories.

Valid examples:
* payments
* checkout
* authentication
* notifications

The value represents an instance, not a type.

Where possible, Zuri resolves the boundary value to known system identity:
* repository identity (`repo_id`)
* service identity from Structure Graph (§17)
* API/service boundary identity (§17.8)

The raw string remains available for human readability, but resolved identity is preferred for retrieval, conflict detection, and cross-repository reasoning.

Example:

```
concern = reliability
decision_type = retry-policy
boundary = payments

resolved_boundary:
  repo_id = <payments-service-repo>
  service_id = <payments-service>
```

#### Decision Key Construction

`decision_key` is derived from normalized classification fields:

```
boundary + concern + decision_type
```

Example:

```
boundary:payments/concern:reliability/decision_type:retry-policy
```

This key identifies the decision area being reasoned about.

It does not attempt to encode organizational ontology. It exists only to provide a stable grouping mechanism for:
* conflicting decisions,
* knowledge gaps,
* retrieval clustering,
* cross-branch decision tracking.

#### Model Calibration Alignment

`model_calibration.extraction_type` is removed.

Extraction calibration is keyed by `concern` because the calibration problem is not the name of the extraction task; it is how reliably a model interprets decisions within a particular engineering area.

Updated schema:

`model_calibration`

| Field | Type | Notes |
|---|---|---|
| `model_id` | text | model provenance identifier |
| `concern` | text | normalized decision concern being calibrated |
| `calibration_curve` | jsonb | mapping from raw confidence to corrected confidence |
| `sample_size` | int | resolved outcomes used for calibration |
| `last_updated_at` | timestamptz | |

This keeps extraction confidence calibration aligned with the same classification system used by memory retrieval and conflict detection.

#### Resulting Invariant

Every decision in Zuri should answer:

1. What engineering concern does this affect?
2. What type of decision is being made?
3. What concrete system boundary does it apply to?

The classification system describes the decision without attempting to become a complete ontology of software engineering.

---

## 8. IDE / Agent Integration

- The primary consumer is the **AI coding agent**, not a human search UI.
- Live context extraction (current file, symbols in scope, prompt text) is the trigger — this determines what memory is relevant for a given generation.
- Retrieval considers both **file/symbol scope** and the **actual prompt text** typed by the human.
- The standalone chat/search UI is not the v1 product — the agent *is* the interface. The Electron GUI (§12) is a management/inspection surface, not a chat interface.
- **MCP config auto-registration**: on first successful GitHub App authorization, Zuri detects installed agent configs on the local machine (`.mcp.json`, `claude_desktop_config.json`, Gemini CLI's equivalent config path, etc.) and writes the Zuri MCP server entry automatically. The user should never hand-edit JSON to connect an agent. If no known config is detected, the GUI shows the config snippet with a "copy" button as fallback.

---

## 9. Retrieval & Ranking

Hardcoded ranking rules ("recency wins" / "canonical always beats probabilistic") are explicitly rejected — the system is too dynamic, and a fixed rule guarantees the wrong outcome for the case that matters most: a quiet, unconfirmed decision that slowly becomes central to the system over months. Importance must be computed continuously, not declared once by a human or a static formula.

### 9.1 Two-Stage Retrieval

- **Stage 1 — Candidate retrieval (cast wide):** semantic similarity (pgvector cosine distance between prompt embedding and `content_embedding`) + structural connection via `memory_touches_file` join against `files_in_scope`. Deliberately generous — this stage's job is to not prematurely exclude the sleeper decision, not to be precise. Target: pull top ~50–100 candidates before ranking.
- **Stage 2 — Ranking (narrow to what fits context):** score every Stage 1 candidate, sort, and truncate to fit `token_budget` from the tool call input.

### 9.2 Latency Target (NFR — resolves the v0.1 latency-vs-on-demand tension)

- **p95 target: under 2 seconds** for combined Stage 1 + Stage 2, for a repo with up to ~5,000 memory records. This is achievable because Stage 1 is a single indexed pgvector query + one indexed join, not a live LLM inference call — the "reconstructed on demand" principle refers to *which memories get selected*, not to generating memory content live. Memory content itself is pre-computed at ingestion time (§10), never generated at query time.
- If p95 exceeds this target during testing, the fix is tightening Stage 1's candidate count or adding a materialized view for `citation_count`/`last_cited_at` — not relaxing the target.

### 9.3 Scoring Model **[AMENDED]**

Importance is **derived, not declared** — modeled as a citation/influence graph (PageRank-style), not a hand-set weight:

- Every PR, commit, or canonical memory entry that references a prior decision raises that decision's centrality score (via `memory_citation` inserts).
- A decision starts as one isolated node; as more PRs/canonical entries cite it, its score climbs *because usage climbs* — nobody has to predict in month one that it'll matter in month six.

Combined with **decay**, not instead of it:

```
score = relevance(prompt) × trend(citation growth over time) × status_weight(tier/status) × recency_of_last_reference
```

- **relevance(prompt)**: cosine similarity from Stage 1.
- **trend**: rate of change in citation count over a rolling window (e.g. citations in the last 30 days vs. the 30 days prior), not raw total count — a decision cited heavily two years ago but untouched since should decay even with a high total count; a decision with few citations but recently accelerating should rise.
- **status_weight**: a lookup table, e.g. `canonical=1.0, probabilistic/confirmed=0.8, probabilistic/proposed=0.6, working=0.4, lapsed=0.1, rejected=0.0` (rejected excluded from retrieval entirely — v1 can hardcode these as a starting default table, tunable per repo later).
- **recency_of_last_reference**: exponential decay function off `last_cited_at`.

**Note per §1.3:** `trend × status_weight`, combined with `citation_count`, is exactly the computation stored as `memory_record.evidence_strength` (§7.2). This section describes how that computation feeds ranking; §7.2 and §13.2 describe how the same computation is also stored and exposed as an independent, named trust signal. There are not two separate formulas — there is one evidence-strength computation, used in two places for two different purposes. `extraction_confidence` deliberately does not appear in this formula — see §1.3 for why, and §10.6/§10.7 for where it's used instead.

### 9.4 Learned Weights (Phase 2 — not v1 blocking)

Rather than hand-setting how much semantic-match should count vs. graph centrality vs. recency, track outcomes: when the agent surfaced memory X and the resulting code merged clean vs. got rejected/reverted in review, that's a training signal for learning-to-rank. **This requires a merge-outcome feedback loop that doesn't exist until the ingestion pipeline (§10) has been live for some weeks** — treat §9.3's fixed formula as the v1 default, and §9.4 as an explicit Phase 2 item once there's outcome data to learn from.

*Relationship to §1.4's model calibration:* both consume outcome data from `audit_log`, but correct different things. §9.4 learns how much each ranking factor (relevance/trend/status/recency) should count, globally. §1.4 learns how much to trust a given model's stated confidence, per model. They can share an outcome-extraction pipeline but produce independent corrections and should not be merged into one learned parameter set.

### 9.5 Revival Check (Lapsed Memory Re-Citation)

Status (including `lapsed`) is a **multiplier in the score**, not just a tag — lapsed memory is never deleted, but its `status_weight` collapses it toward the bottom of ranking by default.

If a lapsed/abandoned memory record suddenly starts accumulating new citations again (trend inverts from flat/decaying to rising), this is treated as a signal: the system writes an `audit_log` entry with `event_type='revival_flagged'` and surfaces a notification to the original proposer/stakeholders — *"this abandoned decision is suddenly being referenced again — should we revisit it?"* Revival is surfaced for a human decision, not resolved automatically in either direction — the record's `status_weight` does not change until a human acts on it via `resolve_memory`.

### 9.6 Cross-Repository Retrieval

A single daemon installation may have multiple repositories connected at once (§12 allows "one repo or a handful of repos connected per install"), which means multi-service integration work, connecting a frontend to a backend across a microservice boundary, for example, requires memory from more than one repository in a single retrieval call. Since all connected repositories on one installation share the same Postgres instance, this does not require federation or a network fan-out; it requires widening the query scope correctly.

**Resolving which repos are implicated.** Every entry in the `get_relevant_memory` input's `files_in_scope` (§13.2) is matched against the `local_path` of every connected repo (§7.2) to determine the full set of repositories this particular call actually touches. A call whose `files_in_scope` spans two connected repos' local paths implicates both; Stage 1 candidate retrieval (§9.1) then queries `WHERE repo_id IN (...)` across that resolved set, not a single repo, and Stage 2 ranking (§9.3) proceeds exactly as already specified, scoring is unaffected by how many repos contributed candidates.

**Integration decisions that belong to more than one repo.** A decision like an API contract or event schema is not naturally owned by a single repository, it describes the boundary between them. Rather than introducing a separate organizational memory tier to hold these, `resolve_memory` (§13.3) accepts an optional array of additional repo IDs a decision should also apply to beyond its originating repository, written to the `memory_applies_to_repo` join table (§7.2) at confirmation time. Retrieval for any of those additional repos then includes this record via the join, alongside records that originate there directly. This means a backend engineer confirming an API contract decision can mark it as also relevant to the frontend repo without either repo needing to duplicate the record or without inventing a broader memory scope than the decision actually warrants.

**This section supersedes the single-repo assumption in §13.2's description of `get_relevant_memory`.** That tool's behavior description should be read as resolving repo scope per this section, not from a single "active project" context.

**At Stage 2 scale (§17.7), the same resolution logic applies across shards rather than within one query.** When repositories are sharded across separate Postgres instances rather than colocated on one machine, the repo-to-shard routing layer described in §17.7 performs the same `local_path`-to-repo resolution, then fans out to each implicated shard and merges results, using the same merge pattern already designed there for combining Decision Memory and Structure Graph. No new merge mechanism is needed at Stage 2, this section's resolution step and that section's fan-out-and-merge step compose directly.

---

## 10. Data Ingestion

### 10.1 Scope

Ingestion is **not** passive scraping of every company tool. It's narrowly scoped to events that already occur at existing engineering gates — for v1, this means **GitHub events only** (Slack/Notion/CI/meetings excluded per §15 Non-Goals). Three ingestion moments map directly onto the memory tiers already defined:

1. **PR merge → canonical memory**
2. **PR open / in-review (RFC-style, not yet merged) → probabilistic memory**
3. **Branch commit activity → working memory**, scoped by `created_by` + `branch_label` (§7.3)

### 10.2 Extraction Method **[AMENDED]**

Hardcoded extraction rules are rejected — PR hygiene is inconsistent (sparse descriptions, "fix bug"-style messages, real reasoning scattered across diff/commits/comments/linked issues), so no fixed parser can reliably pull out "the decision." Instead: **an AI model infers** the candidate decision from the full PR context (diff + description + commit messages + review comments + linked issues), the same way a thoughtful reviewer would piece it together manually.

Inference is never trusted silently — it surfaces as a **bot comment directly on the PR itself**, next to the code the human is already reviewing: *"Zuri inferred this decision from this PR: [X]. Confirm, edit, or reject?"* Reviewer replies map to the `resolve_memory` tool (§13.2) via a GitHub webhook listener on comment events. This keeps memory-writing on the exact same gate as code review (§4) — no separate approval workflow for reviewers to skip or forget.

**Confidence emission [NEW]:** the extraction prompt requires the model to return a structured `extraction_confidence_raw` (0–1) alongside the inferred `decision` and `reasoning`, and a `concern` classification (e.g. `reliability`, `data`, `security`) matching the categories `model_calibration` (§7.2, §7.4) tracks separately. The raw value is corrected against that model's calibration curve for that concern before being written to `memory_record.extraction_confidence`. Below a minimum `sample_size` in `model_calibration` (a new model/concern pairing with no track record yet), the raw value passes through uncorrected and is flagged as uncalibrated in the audit log, rather than silently applying a curve with no statistical basis.

**Merge state signaling**: when the merge webhook handler calls `resolve_memory` internally (either directly or via the bot-comment-reply path), it must pass the `pr_merged` field explicitly per §13.3's schema. `pull_request.closed` with `merged:true` maps to `pr_merged: true`; every other path maps to `pr_merged: false` or omits it. This must be a structural field on the call, never inferred by string-matching `source_context` or any other free-text value.

**Model choice for extraction**: this runs server-side inside the Go daemon, calling out to an LLM API (model-agnostic — configurable per install, default to a fast/cheap model since this is a background inference task, not a user-facing chat). This is a separate concern from which agent/model the *developer* is using in their IDE. Per §1.2, the chosen `model_id` is recorded on every extracted record as provenance only.

### 10.3 Pipeline Mechanics

- **Read-only GitHub App** (not a personal access token, not full OAuth) — scoped to `pull_requests:read`, `contents:read`, `issues:read`, `metadata:read`. No write scope for repo content in v1: Zuri cannot modify the codebase, only read it. (Zuri *does* write PR comments, which requires `pull_requests:write` narrowly for the comment API only — this is a materially smaller trust ask than modify-code access, and should be called out explicitly as such during GitHub App manifest registration and in any user-facing permissions explanation.)
- **Webhook-driven, not polling** — GitHub pushes events (`pull_request.opened`, `pull_request.synchronize`, `pull_request.closed` with `merged:true`, `pull_request_review_comment.created`, `issue_comment.created` for bot-comment replies) the moment they happen.
- **Webhook receiver lives inside the same Go daemon** as the MCP server — one process, one route (e.g. `POST /webhooks/github`), verified via GitHub's webhook signature (HMAC secret configured at App registration).

```
GitHub webhook fires (PR opened/updated/merged/commented)
      ↓
Zuri fetches PR diff + description + comments + linked issues (read-only)
      ↓
Extraction model infers candidate decision(s) + extraction_confidence_raw + concern
      ↓
Posted back as a bot comment on the PR itself
      ↓
Human reviewer confirms/edits/rejects via comment reply → parsed → resolve_memory called internally
      ↓
On PR merge: confirmed candidates write to memory_record (tier depends on PR state — canonical if merged to main, probabilistic if still open/RFC-style)
```

### 10.4 Silence Is Not Rejection — the Persistent Pending Rule

**An unconfirmed AI-inferred decision must never be silently discarded, and it must never expire quietly.** If a reviewer merges a PR without engaging the bot comment, the underlying code change still lands — memory must not fall behind it.

Rule: an unconfirmed extraction tied to a **merged** PR is written immediately as `tier=probabilistic, status=proposed` (never discarded, never silently promoted to canonical either) and is then **actively re-surfaced, not passively left to expire** — it keeps getting pinned back to the responsible reviewer/team until it is explicitly confirmed or explicitly rejected. This is distinct from ordinary lapsing in §6: ordinary probabilistic memory expires because its proposal window closed with no resolution either way and quietly downgrades; a merged-but-unconfirmed decision keeps actively demanding resolution because the code is already live and the brain's gap against it is a standing risk, not a background one.

**Re-surfacing cadence — hybrid, fixed floor + configurable layer:**

- **Fixed default (always on, not disable-able below a floor):** a time-based reminder cadence ships out of the box (default 7 days, per `zuri_config.reminder_cadence_days`), so no org can silently let a merged-but-unconfirmed decision go unaddressed indefinitely.
- **Event-triggered (always on):** the moment another PR touches the same files/symbols/decision area (checked via `memory_touches_file` overlap), the unconfirmed record is re-flagged immediately, independent of the time-based cadence.
- **Org-configurable layer on top:** the org can tune the time-based interval and add extra notify channels (`zuri_config.additional_notify_channels` — Slack webhook etc., stubbed field in v1 schema, not implemented in v1 logic). What can't be configured away is the floor itself — reminders can be more frequent or routed differently, never disabled outright.

*Distinction from §10.7:* this section governs merged-but-unconfirmed decisions — memory that exists but hasn't been reviewed. §10.7 governs knowledge gaps — the absence of any reliable decision at all. Both use similar re-surfacing instincts but are different entities (`memory_record` vs. `knowledge_gap`) with different cadences (`reminder_cadence_days` vs. `gap_digest_cadence_days`).

### 10.5 Greenfield Onboarding Ingestion **[AMENDED]**

Sections 10.1 through 10.4 assume PR history exists to extract from. A company on day one has none. The only source of truth at that point is what the founder or founding team can articulate directly, so this requires a distinct ingestion path, not a variant of the PR-based pipeline.

**Mechanism**: an adaptive intake flow, run from the Electron GUI's "Create Brain" wizard, not through the MCP tool surface, since this is a founder-facing setup action, not an agent-facing runtime action, and keeping the MCP tool surface agent-only (per section 13) avoids conflating two different kinds of caller. Each answer determines the next question rather than presenting a fixed form, covering domain, expected scale, which tradeoffs the founder actually cares about, hard constraints such as compliance or data residency, and explicit non-goals. The exact question tree is an implementation detail to design when this phase is opened, not specified here.

**Write path and the gate it substitutes for**: section 4 requires every write to canonical memory to pass through a review gate equivalent to code review, no exceptions. There is no PR to attach this to, so the review moment is the founder explicitly confirming the survey's synthesized output before it is written, screen by screen or as a final summary confirmation, functioning as the same kind of deliberate approval a PR merge represents, just without a diff attached to it.

**Schema fields used**: `source_type='onboarding_survey'`, defaulting existing rows to `pr_merge` for backward compatibility. `model_id` and `extraction_confidence` are left null for these records — there is no artifact being interpreted, only a direct human assertion, so extraction confidence as defined in §1.3 doesn't apply. **`evidence_strength` for onboarding-derived memory is seeded at a fixed moderate baseline** (config constant, not zero and not maximum) rather than left at whatever a first-citation record would compute to — a founder's stated intent is real evidence, but weaker than a subsequently-confirmed, PR-backed decision, and should be treated that way from the start rather than starting at zero and requiring citations to "catch up" to a status it arguably already deserves. This baseline is a judgment call, not derived from any formula, and should be tunable per install.

**No special-cased decay or expiry**: onboarding-derived memory participates in the same scoring model as any other canonical record (section 9.3), with no separate expiry logic. As real PR-derived canonical memory accumulates on the same topic and gets cited, it naturally outranks stale founding assumptions through the existing trend and recency mechanics, no new code path is needed for founding memory to fade in relevance as the company's real decisions accumulate around and eventually past it.

The Electron GUI's record inspector (section 12) should render onboarding-sourced records with a visibly distinct marker, for example a "founding intent" tag alongside the existing tier badge, since a founder's stated intention and a team's proven-in-production decision are different weights of truth even when both carry `tier=canonical`.

### 10.6 Backfill Prioritization for Existing Codebases **[AMENDED]**

Running the full extraction pipeline (section 10.2) against years of historical PRs is expensive in LLM tokens and, at real scale, not worth doing exhaustively. Uniform backfill is the wrong default; prioritized backfill against an explicit budget is correct regardless of how much processing time or cost is available, because processing low-impact history at the same priority as high-impact history is not a more thorough approach, it is a worse one.

**Prioritization mechanism**: this depends on Structure Graph (section 17) to be fully realized, since centrality data is what makes prioritization meaningful rather than arbitrary. Once Structure Graph exists: run its deterministic, non-LLM bootstrap pass (section 17.5) first, this produces a call and dependency graph with no inference cost. Use centrality within that graph, files and modules with the most incoming structural connections, to rank historical merged PRs by how many high-centrality files they touched. Feed the extraction pipeline (section 10.2) that ranked list, highest-centrality first, recency as a secondary sort within equal centrality, rather than chronological or arbitrary order.

**Until Structure Graph exists**: backfill prioritization falls back to recency-weighted order alone, most recent merged PRs first, since centrality data is not yet available to rank by structural importance. This is an explicit interim behavior, not a placeholder to silently forget about once section 17 is built.

**Confidence-based review triage [NEW]:** at organizations large enough that backfill produces thousands of candidate extractions, uniform human review of every bot comment is not realistic even with prioritized *ingestion* order — a separate triage question is which already-extracted candidates most need a human's limited attention. `extraction_confidence` (§1.3, §10.2) is the signal for this: candidates below a configurable confidence threshold are queued for review first, since these are the ones most likely to contain a misreading a reviewer would actually catch, while high-confidence, high-evidence-strength candidates can be safely left in the standard persistent-pending flow (§10.4) rather than pulled into a priority queue. This is the review-priority use of extraction confidence anticipated in §1.3 — it never affects retrieval ranking, only where a human's attention is directed during high-volume backfill.

**Budget, not completeness, is the design target**: the org sets an explicit backfill budget in `zuri_config` (`backfill_budget`, §7.2), either a token cap or a PR count cap. Processing stops when the budget is exhausted, not when history runs out. This means backfill is deliberately incomplete by design for any codebase with enough history to exceed the budget, the long tail of low-impact historical PRs may never be processed, and that is a stated tradeoff, not a defect. Every skipped PR should be logged (which PR, why it was skipped, that it was a budget cutoff rather than an error) so the gap is visible and inspectable, never silently absent.

### 10.7 Knowledge Gap Detection **[NEW]**

Sections 10.2–10.6 describe extracting memory when evidence exists. This section describes what happens when it doesn't — the mechanical implementation of §1.5's Knowledge Recovery principle.

**Detection triggers.** A `knowledge_gap` row is created when any of the following occur during extraction, backfill, or retrieval:

- **Conflicting conventions**: two `memory_record` rows share a `decision_key` but assert incompatible values (§1.3's conflict definition, structural not semantic). Both records are linked via `affected_memory_ids`; `gap_type='conflicting_conventions'`.
- **Insufficient evidence**: extraction produces a candidate decision with both low `extraction_confidence` and low resulting `evidence_strength` (i.e., neither the model was sure, nor did subsequent activity reinforce it) that a reviewer neither confirmed nor rejected before its `expires_at` — rather than simply lapsing per §6, if no `decision_key` peer exists to inherit context from, it's logged as a gap so the absence is visible, not just a quietly aging `lapsed` row indistinguishable from an abandoned-but-once-good idea.
- **Unowned decision**: a `decision_key` is referenced (via `memory_touches_file` overlap on active work) with no existing `memory_record` at all — an agent or extraction pass hit an area with no memory and nothing to surface.
- **Stale unreinforced**: a canonical memory's `evidence_strength` has been monotonically declining (no citations, no reinforcing changes) past a configurable threshold — flagged for review rather than silently continuing to decay, since stale-and-forgotten and stale-and-superseded look identical from decay alone and only a human can distinguish them.

**Resolution.** A gap moves to `answered` when a human provides a decision through the GUI or via a reply to a surfaced digest item; this creates or updates the linked `memory_record` at the moderate baseline evidence strength described for onboarding records in §10.5 (an answer is real evidence, not maximal evidence — it strengthens as subsequent work conforms to it, same as any other signal). A gap moves to `acknowledged_unknown` when a human explicitly confirms the absence is intentional — this is itself recorded as evidence (`audit_log`, `event_type='gap_acknowledged_unknown'`) and the gap is not re-surfaced again unless a *new* conflicting or reinforcing signal appears later, at which point it reopens.

**Surfacing at scale.** Naive one-gap-one-interrupt notification fails once an organization has more than a handful of teams: the same underlying gap (a missing retry-policy convention, say) can appear independently across dozens of services, and interrupt-per-occurrence trains people to ignore the system. Surfacing therefore requires:

- **Ownership-based routing** (`knowledge_gap.routed_to`): derived from the repository's `CODEOWNERS` file (already readable under the existing `contents:read` scope, §10.3 — no new GitHub App permission required) and recent commit authorship on the affected files, falling back to `zuri_config.approver_usernames` when neither signal is available. Never routed to an undifferentiated leadership inbox.
- **Clustering before surfacing**: gaps sharing a `decision_key` and overlapping affected scope are merged into a single surfaced item listing every affected location, not raised once per occurrence. This is the single highest-leverage mechanism at scale and depends entirely on `decision_key` existing and being normalized per §7.4.
- **Priority ranking**: by blast radius (`affected_memory_ids` count and downstream `memory_touches_file` overlap), by live friction (an agent hitting this exact gap mid-session with nothing to inject, logged via `audit_log` at retrieval time, is a stronger and more current signal than any static heuristic), and by recency of related activity.
- **Batched delivery**: default delivery is a digest, on `zuri_config.gap_digest_cadence_days` (default 7) or at onboarding time for a newly connected repo/team, surfaced in the GUI's inspector (§12) — not a real-time interrupt. Real-time surfacing is reserved for a high-blast-radius gap encountered live during an active session. `additional_notify_channels` (Slack, etc.) remains the same unimplemented-in-v1 stub already defined in §7.2; v1 delivery is GUI-only.

---

## 11. Storage Architecture — Postgres + pgvector Only (No Graph DB)

**Decision, closed**: a single Postgres database with the pgvector extension is the entire storage layer. No Neo4j, no Apache AGE, no second database of any kind in v1.

**Why**: the only traversal need identified — "what files does this memory touch, what other memories touch those same files" — is 1–2 hop traversal, not deep multi-hop graph querying. This is handled by the `memory_touches_file` join table (§7.2) with plain joins or recursive CTEs if traversal depth needs increase later. PageRank-style centrality (§9.3) is computed as a scheduled batch job over `memory_citation`, which is an adjacency structure Postgres represents natively — a dedicated graph engine is not required to compute centrality scores.

**Rejected alternative and why**: introducing Neo4j means a second database to deploy, back up, and keep in sync (dual-write or event-driven sync, each with its own failure modes), and — critically — it breaks the single-binary packaging goal (§12), since Neo4j is a JVM server process that cannot be compiled into one executable. Bundling a JVM inside the Electron installer, or depending on a hosted Neo4j Aura instance, both contradict the "download one EXE, run it, done" distribution model this project is built around. Revisit only if query complexity or traversal depth becomes a measured bottleneck post-launch — not before, and not speculatively.

**Embedded vs. server Postgres for distribution**: for v1, ship an embedded/managed Postgres instance (e.g. via a bundled Postgres binary the Go daemon spawns and manages, or SQLite with a migration path to Postgres if embedding proves too heavy for the installer size target — **this specific sub-choice is left to implementation discretion**, since it doesn't affect the schema or any decision above, only how the daemon starts up).

---

## 12. Packaging & Distribution **[AMENDED]**

**Language decision**: the daemon is written in Go, not TypeScript/Bun as considered earlier. Reasoning, recorded here so it isn't re-litigated later: Go compiles to a truly static binary with no bundled runtime, which is a better fit for the single-executable distribution goal than a Bun-compiled binary (which still carries the JS runtime inside it). Go's official MCP SDK (`github.com/modelcontextprotocol/go-sdk`, maintained in collaboration with Google) is spec-complete and supports Streamable HTTP transport, closing the maturity gap that previously favored TypeScript. Goroutines are also a better structural fit for the daemon's actual workload, concurrent webhook handling plus the scheduled re-surfacing job (§10.4) plus batch PageRank-style scoring (§9.3) are background-concurrency-heavy, not purely I/O-bound request handling. The Electron GUI remains TypeScript/JS, since Electron requires it, this is unavoidable and does not conflict with the daemon's language, since the two are separate processes communicating over localhost HTTP regardless of what either is written in.

**Architecture, Docker-Desktop-style**: an Electron GUI shell wraps a Go-compiled background daemon. Same split as Docker Desktop (Electron GUI) wrapping the Docker daemon.

- **Go-compiled binary** = the Brain daemon: embedded Postgres/pgvector, GitHub webhook receiver, MCP server (Streamable HTTP transport — §13.1), SSE endpoint for GUI live activity (§13.3), extraction/ranking logic.
- **Electron app**, packaged via `electron-builder` into a single installer per OS (`.exe` / `.dmg` / `.AppImage`) = the GUI shell. Spawns the Go daemon as a child process on launch, communicates over localhost.

**First-run flow:**
1. User downloads the open-source build from the website → runs the installer → single executable, no separate dependencies to install.
2. GUI opens → "Connect a project" → GitHub App install flow (browser popup, read-only scopes + narrow PR-comment write scope per §10.3).
3. On authorize, Zuri **auto-writes the MCP config entry** into detected agent configs on the machine (§8) — no manual JSON editing.
4. Daemon starts listening; agents are live immediately for that repo.

**GUI structure (the "Docker containers list" analogy):**
- Left sidebar: list of connected repos/projects (the "containers").
- Click into a project → view its memory records: canonical / probabilistic / working, filterable by tier and status, with lapsed/revival flags visibly surfaced (not buried).
- Per-project toggle: turn a project's connection on/off (stops the daemon serving that repo's memory to agents; does not delete data — same non-destructive principle as §6).
- Per-record inspector: click a single memory record to see full history — decision text, reasoning, source PR, citation count/trend, resolution history, **and per §1.3, its extraction confidence and evidence strength shown as two separate values with a short rationale, not one blended score** — this is the "`docker inspect`" equivalent.
- **Knowledge gap inbox [NEW]:** a distinct view from the memory record list, showing open/surfaced gaps per §10.7, filterable by status, with an explicit "acknowledge as unknown" action alongside "answer this."
- Settings pane per project: edit `zuri_config` values (approvers, expiry window, reminder cadence, backfill budget, gap digest cadence) — this is the GUI surface for §5's configurability.
- Live activity feed: real-time stream (via the SSE endpoint, §13.3) showing "Agent X just queried memory for prompt Y" as it happens.

**GUI is a management/inspection surface only — it is not the memory itself and not the primary product interface.** The MCP server + agent integration is the primary product; the GUI exists so a human can see what the daemon is doing, same as Docker Desktop exists to inspect a system that runs fine headless.

---

## 13. Transport & MCP Tool Contracts

### 13.1 Transport Decision

MCP defines two official transports — stdio (host process spawns the server directly, pipes over stdin/stdout) and Streamable HTTP (server runs as a long-lived process on a local port; can use SSE for server-initiated push, not for streaming tool-call responses token-by-token). Both are valid ways to build an MCP server; the tool-handling logic is transport-agnostic in the standard SDKs. Implementation uses the official Go SDK (`github.com/modelcontextprotocol/go-sdk`), which supports both transports and is spec-complete, so switching from the originally considered TypeScript SDK loses no transport capability.

**Decision: Streamable HTTP for the MCP server itself.** Reason: the daemon is long-running and GUI-managed rather than spawned per-agent-session, and multiple agents (e.g. Claude Code and Gemini CLI open simultaneously on the same machine) need to hit **one running daemon instance** rather than each spawning their own server process.

**Decision: a separate SSE endpoint, not on the MCP tool-call path, purely for the GUI's live activity feed** (§12). This is a legitimate server-push use case — pushing "agent X just queried Y" events to the Electron UI in real time — distinct from and not to be confused with the MCP tool-call request/response, which is single-shot: the retrieval computation (Stage 1 + Stage 2, §9) completes server-side before a response is returned, so there is no meaningful partial result to stream mid-computation on that path.

**Testing Requirement**: An explicit end-to-end test requirement MUST exercise the actual Streamable HTTP server by POSTing/requesting through the registered `http.ServeMux` rather than invoking handlers directly. This prevents HTTP routing regressions (such as route handler shadowing/overrides) that unit tests against handler functions alone would miss, complementing the transport requirements without altering the underlying architecture.

### 13.2 Tool: `get_relevant_memory` **[AMENDED]**

**Input schema:**
```json
{
  "prompt_text": "string — the developer's actual prompt, required",
  "files_in_scope": ["array of strings — file paths currently open/touched, required, may be empty array"],
  "token_budget": "integer — how much context room the agent has left, required"
}
```

**Output schema:**
```json
{
  "query_tokens_used": "integer — approximate token cost of the returned payload",
  "memories": [
    {
      "memory_id": "string (UUID)",
      "tier": "canonical | probabilistic | working",
      "status": "confirmed | proposed | lapsed | rejected",
      "decision": "string — the decision text",
      "reasoning": "string — supporting context",
      "source": {
        "pr": "integer, nullable",
        "merged_at": "ISO timestamp, nullable",
        "state": "string, e.g. 'merged' | 'open', nullable"
      },
      "touches": ["array of file path strings"],
      "confidence": {
        "extraction_confidence": "float 0-1, nullable — nullable for onboarding-derived memory, see §1.3/§10.5",
        "evidence_strength": "float 0-1",
        "rationale": "string — short, human-readable summary of why this score is what it is, e.g. 'confirmed via PR review, cited 6 times in last 30 days'"
      },
      "score": {
        "relevance": "float 0-1",
        "trend": "string — 'rising' | 'flat' | 'decaying'",
        "citations": "integer",
        "last_cited": "ISO timestamp"
      }
    }
  ]
}
```
*New `confidence` block per §1.1/§1.3: exposes the two independent trust assessments to the calling agent, distinct from the existing `score` block, which governs ranking and remains unchanged in meaning. `rationale` is intentionally a short string rather than a structured breakdown — enough for a human or agent to sanity-check the number without requiring them to parse the full scoring formula.*

**Behavior**: executes Stage 1 candidate retrieval + Stage 2 ranking (§9) against the repo or repos implicated by `files_in_scope`, resolved per §9.6 rather than a single fixed "active project" (this description previously assumed one active repo; §9.6 supersedes that). Truncates `memories[]` to fit `token_budget`. Every call writes an `audit_log` entry with `event_type='retrieved'`.

### 13.3 Tool: `resolve_memory`

Two possible callers: the GitHub bot-comment flow (parsed reply on a PR) and the coding agent directly (a developer tells their agent mid-session to confirm/reject a decision it just surfaced).

**Input schema:**
```json
{
  "memory_id": "string (UUID), required",
  "action": "confirm | reject | edit, required",
  "edited_content": "string — required only when action is 'edit'; replaces 'decision' before confirming",
  "resolved_by": "string — GitHub username or agent-session identity, required",
  "pr_merged": "boolean, optional — explicit flag indicating whether the originating PR has merged to the default branch, default false",
  "applies_to_repo_ids": "array of strings (UUID), optional — additional repos beyond this memory's originating repo that this decision is also explicitly relevant to, written to memory_applies_to_repo per §9.6, used for cross-service integration decisions such as API contracts",
  "source_context": "string, optional — e.g. 'PR #601 comment' or 'agent session: <prompt excerpt>', purely for the audit trail, never parsed or used to drive control flow"
}
```

`pr_merged` is the sole signal used to decide tier transition on `confirm`. It must never be inferred by parsing `source_context` or any other free-text field; `source_context` is logged verbatim to `audit_log.context` for human review and has no bearing on control flow. Defaulting `pr_merged` to `false` when omitted keeps the safe outcome (probabilistic, not canonical) the default rather than requiring every caller to explicitly opt into the conservative path.

**Output schema:**
```json
{
  "memory_id": "string (UUID)",
  "previous_status": "string",
  "new_status": "confirmed | rejected | proposed",
  "new_tier": "canonical | probabilistic | discarded",
  "resolved_at": "ISO timestamp"
}
```

**Behavior rules** (ties to §4/§10.4):
- `confirm` with `pr_merged=true` → `tier` → `canonical`, `status` → `confirmed`, permanent, `expires_at` cleared. `evidence_strength` is recomputed immediately to reflect the confirmation event, not left at its pre-confirmation value.
- `confirm` with `pr_merged=false` or omitted → stays `tier=probabilistic`, `status=confirmed`, `expires_at` set per `zuri_config.expiry_days` from `resolved_at`.
- `reject` → `status=rejected`, record retained (not deleted), excluded from active ranking (`status_weight=0` per §9.3). This outcome is also fed to `model_calibration` (§1.4) as a negative signal for the model/concern pairing that produced it.
- `edit` → `decision` field is overwritten with `edited_content` before the `confirm` logic above applies — this is the mechanism that prevents a bad AI inference from calcifying into canonical truth. Also fed to `model_calibration` as a partial-miss signal, distinct from a clean confirm or a clean reject.
- This tool **can only transition an existing `proposed` record** — it has no code path to create a new `confirmed` canonical record directly, mechanically enforcing §4.
- Every call writes an `audit_log` entry with `event_type` matching the action taken (`confirmed`/`rejected`/`edited`), regardless of outcome.

### 13.4 Tool: `resolve_knowledge_gap` **[NEW]**

Parallel to `resolve_memory`, but for entities in `knowledge_gap` rather than `memory_record`. Two possible callers: a human responding to a surfaced digest item via the GUI (§12), or a coding agent relaying a developer's answer mid-session when it encounters an open gap.

**Input schema:**
```json
{
  "gap_id": "string (UUID), required",
  "action": "answer | acknowledge_unknown, required",
  "answer_content": "string — required only when action is 'answer'; becomes the decision/reasoning for a new or updated memory_record",
  "resolved_by": "string — GitHub username or agent-session identity, required"
}
```

**Behavior:**
- `answer` → creates or updates a `memory_record` linked via the gap's `decision_key`, at the moderate baseline `evidence_strength` described in §10.7, and sets `knowledge_gap.status='answered'`.
- `acknowledge_unknown` → sets `knowledge_gap.status='acknowledged_unknown'`, does not create any `memory_record`. The gap is not re-surfaced unless a new conflicting or reinforcing signal later reopens it (§10.7).
- Every call writes an `audit_log` entry with `event_type` matching the action (`gap_answered` / `gap_acknowledged_unknown`).

---

## 14. Audit Logging & Access Model

- **v1 scope: audit logging is fully implemented; role-based access control (RBAC) enforcement is DEFERRED.**
- Every read (`get_relevant_memory`) and every write (`resolve_memory`, `resolve_knowledge_gap`, ingestion pipeline events, lapse events, revival flags, gap detection/surfacing events) writes an append-only row to `audit_log` (§7.2). This alone gives full traceability: who/what queried which memory, when a memory was created/confirmed/rejected/edited and by whom, and — per §1.1 — why the system believed it, since the same log rows that drive `model_calibration` (§1.4) are also the "why" record a human can inspect.
- **Why RBAC enforcement is deferred, not skipped**: v1 is architected as a single-daemon-per-developer-machine model (one Electron app instance per user, connected to whichever repos that user has GitHub access to). GitHub's own repo permissions are the de facto access boundary in v1 — Zuri does not need to reimplement permission logic when GitHub already gates who can see a private repo's PRs in the first place. Formal in-app RBAC (e.g. "this teammate can view but not confirm memory") becomes a real requirement only at multi-seat/team-shared-daemon scale, which is explicitly out of v1 scope.
- **This must be revisited before any multi-seat/shared-instance deployment** — flag this explicitly in code comments near the daemon's auth boundary so it isn't quietly assumed to be "handled."

---

## 15. Non-Goals for v1

- Full ingestion of Slack, Notion, CI/CD, historical IDE telemetry — later, not v1.
- Auto-generated architecture diagrams — research-stage, not a v1 feature.
- "Team memory" as an always-on profiling system — reframe as ownership/expertise routing if pursued at all; avoid anything that reads as developer surveillance.
- Enterprise features (SSO, SOC2, on-prem) — post-PMF, not a planning input now.
- Neo4j/graph database of any kind — see §11, closed decision for Decision Memory. Structure Graph, a separate deferred subsystem, uses Postgres plus Apache AGE instead — see §17, not in current build scope.
- Structure Graph and the Inference Layer (universal code structure graph, tree-sitter-based parsing, cross-language call graphs) — a named, deferred third subsystem, fully scoped in §17. Not in current build scope. Do not build any part of §17 unless it is explicitly reopened as a new phase.
- Learned/adaptive ranking weights (§9.4) — v1 ships the fixed formula in §9.3; learning-to-rank is Phase 2, contingent on outcome data existing.
- In-app RBAC — see §14, deferred to multi-seat phase.
- Streaming/SSE on the MCP tool-call path — see §13.1; SSE is used only for the GUI activity feed, not tool responses.
- **[NEW]** Slack/external notify delivery for knowledge gap digests — `additional_notify_channels` remains an unimplemented v1 stub; v1 gap surfacing is GUI-digest-only per §10.7.

---

## 16. Build Order (Suggested) **[AMENDED]**

This is a suggested sequence for a coding agent to follow, front-loading the pieces other pieces depend on:

1. **Decision classification taxonomy (§7.4)** — implement controlled `concern` & `decision_type` enums and `boundary` resolution.
2. **Postgres schema** (§7.2) — all tables, indexes, the `memory_citation`-driven trigger for `citation_count`/`last_cited_at`, the `evidence_strength` materialized computation, and the `model_calibration`/`knowledge_gap` tables.
3. **Go daemon skeleton** — process that boots, manages embedded Postgres, exposes health check.
4. **MCP server, Streamable HTTP transport** (§13.1) — wire up `get_relevant_memory`, `resolve_memory`, `resolve_knowledge_gap` (§13.2/13.3/13.4) against the schema, even with stubbed/naive scoring initially. Must include end-to-end HTTP integration tests that request through the registered `http.ServeMux` (rather than isolated handler function calls) to ensure routing correctness and prevent route handler shadowing.
5. **Scoring model** (§9.3) — replace naive scoring with the real formula once retrieval plumbing works end-to-end.
6. **GitHub App registration + webhook receiver** (§10.3) — manifest, signature verification, event routing.
7. **Extraction pipeline** (§10.2) — LLM inference call including `extraction_confidence_raw`/`concern`, bot comment posting.
8. **Bot-comment-reply → `resolve_memory` parsing** (§10.3 diagram) — closes the ingestion loop.
9. **Persistent pending / re-surfacing logic** (§10.4) — scheduled job + event-triggered re-flagging.
10. **Knowledge gap detection + surfacing** (§10.7) — leverages normalized `decision_key` (§7.4) for clustering/conflict detection.
11. **Model calibration** (§1.4, §7.4) — begins accumulating data as soon as step 8 starts producing confirm/edit/reject outcomes, keyed by `(model_id, concern)`.
12. **Electron GUI shell** (§12) — project list, per-record inspector (with confidence/evidence-strength shown separately), knowledge gap inbox, settings pane, live activity feed (wire to SSE endpoint, §13.1).
13. **Auto MCP-config registration on first auth** (§8).
14. **Audit log surfacing in GUI** (§14) — not required for daemon correctness, but needed before this is usable by a non-technical stakeholder per §1's premise.

## 17. Structure Graph & Inference Layer (Deferred Phase — Not Current Build Scope)

Everything in sections 1 through 16 describes one of three subsystems Zuri is ultimately composed of. Only the first is built. The other two are named and scoped here so they are not lost, and so nothing in the current build accidentally forecloses them, but no part of this section is in scope until explicitly reopened as a new phase.

### 17.1 The Three Subsystems

- **Decision Memory** (built, S1 through S3): the canonical/probabilistic/working tiers described in sections 3 through 14. Postgres plus pgvector. Answers "why is this the way it is." PR-gated per section 4.
- **Structure Graph** (deferred): a language-agnostic structural map of the codebase itself. Answers "what exists and how is it connected," not why. Deterministic, not inferred from prose the way Decision Memory extraction is.
- **Inference Layer** (deferred): the AI-driven subsystem that resolves structural ambiguity the parser alone cannot, and places that resolved meaning onto the Structure Graph. Distinct from the extraction pipeline in section 10.2, which infers decisions from PR text for Decision Memory. The Inference Layer infers structural meaning from code itself, for example recognizing that a given module functions as the actual payment boundary even though no single line of code states that.

*Forward note per the confidence model (§1.3):* Structure Graph's `Decision` node type (§17.3) is a natural future home for the normalized `decision_key` taxonomy (§7.4) — a decision key resolves to a `Decision` node once Structure Graph exists, giving conflict detection and gap clustering a real graph identity rather than a flat string field. This is noted for forward compatibility only; it does not change §17's deferred status or pull any of this into current scope.

### 17.2 Structure Graph Storage

Structure Graph uses the same Postgres instance as Decision Memory, extended with **Apache AGE**, a Postgres extension providing native graph storage and Cypher-style query support, rather than a second database engine such as Neo4j. This preserves the single-binary packaging goal from section 12, since AGE is an extension within the same Postgres process, not a separate JVM server. This is an explicit, revisitable choice: if Postgres plus AGE proves insufficient for call-graph traversal depth or performance at scale, the graph model and Cypher-style query patterns are intended to port to a dedicated graph database such as Neo4j with less rework than if traversal logic had been hand-written as Postgres recursive CTEs from the start. Actual data migration and the packaging tradeoff (Neo4j reintroduces a JVM dependency) would still apply at that point; only the query logic itself is intended to carry over cleanly. Postgres extension version compatibility between AGE and whatever Postgres major version the embedded daemon runs must be explicitly verified when this phase is built, not assumed.

### 17.3 Universal Node and Relationship Model

Structure Graph models software engineering concepts, not language syntax, so the same graph schema applies regardless of what language a given repository is written in. Language-specific parsers translate source code into this universal representation; the graph schema itself never changes per language.

Node types fall into three groups:

- **Project structure**: Organization, Project, Repository, Service, Module, Package, File, Function, Class, Interface, API, Database, Queue, Cache, External Service, Environment.
- **Engineering knowledge**: ADR, RFC, PRD, Issue, Pull Request, Commit, Deployment, Bug, Feature, Decision, Requirement, Test.
- **Human knowledge**: Team, Developer, Owner, Reviewer.

Relationships connect these nodes, for example: Repository CONTAINS Service, Service CONTAINS Module, Module CONTAINS Function, Function CALLS Function, Service DEPENDS_ON Service, Service USES Database, Service EXPOSES API, Issue FIXED_BY Commit, Commit MODIFIED File, Developer OWNS Service, ADR JUSTIFIES Decision, Decision AFFECTS Service, Deployment DEPLOYED Service, Bug RELATED_TO Feature. This list is illustrative, not exhaustive; the schema is expected to grow as this phase is built out in detail.

### 17.4 Language Parsing

Parsing is built on **tree-sitter**, which provides production-grade grammars for every language Zuri needs to support (Go, Rust, Python, Java, C#, JavaScript, TypeScript, C++, and others). Tree-sitter supplies the parse tree; Zuri's own logic is responsible for translating that parse tree into the universal node and relationship model in section 17.3. This is stated explicitly because "write a parser per language" understates the actual work; tree-sitter removes the burden of parsing itself, it does not remove the burden of correctly mapping parsed structure onto Zuri's universal schema, including cross-file and cross-service call resolution, which is nontrivial regardless of the parsing library used.

### 17.5 Ingestion Cadence

A one-time structural snapshot is insufficient on its own: code structure changes with every merge, so a graph built once and never updated would drift from the real codebase almost immediately and become actively misleading rather than merely incomplete. A purely continuous full-repository reparse on every event is also incorrect, not merely inefficient, since only a small, identifiable part of the codebase changes with any single merge.

The correct cadence, therefore, is both:

- **One-time bootstrap pass**: when a repository first connects to Zuri, a full parse builds the initial Structure Graph from scratch.
- **Incremental, diff-scoped updates thereafter**: triggered by the same GitHub merge webhook already driving Decision Memory ingestion in section 10.3, reparsing only the files touched by that PR's diff, not the full repository. This reuses the existing webhook infrastructure rather than standing up a second event pipeline; one webhook event, two subsystems reacting to it independently.

### 17.6 Inference Layer Placement

The Inference Layer runs as part of the same incremental update described in section 17.5, not as a separate scheduled process. When diff-scoped structural changes are parsed, the Inference Layer resolves any structural ambiguity the parser cannot (for example, identifying architectural roles like "this is the payment boundary" that are not literally stated in code) and writes the resolved result onto the Structure Graph at that point. It does not run continuously independent of code changes, and it does not run as a one-time pass independent of the bootstrap and incremental cadence above.

### 17.7 Combined Retrieval and Horizontal Scaling

This section designs how Structure Graph and Decision Memory are queried together, and how that combined retrieval scales horizontally as the number of connected repositories and organizations grows. This is explicitly a Stage 2 design, consistent with the roadmap noted elsewhere in this document; nothing here contradicts the single-daemon, single-machine model that sections 1 through 16 are built against, it describes what replaces that model when this phase is opened, not a change to the current build.

**Unified query, not two result sets to reconcile.** The agent-facing contract stays a single tool call returning a single ranked JSON payload, the shape already defined in section 13.2 is not broken by this design. Internally, `get_relevant_memory` fans out to both subsystems in parallel: Decision Memory runs its existing Stage 1 and Stage 2 retrieval (section 9), Structure Graph runs a traversal query scoped to the same `files_in_scope` input, using AGE's Cypher-style querying to find structurally adjacent code, callers, dependents, and shared modules. A merge step combines both into one ranked list before the tool returns, the calling agent never has to separately query or reconcile two subsystems itself.

**How structural signal feeds the existing score, rather than living beside it.** Structure Graph's contribution is an additional signal folded into the same section 9.3 formula, not a competing parallel score. A candidate decision whose touched files are structurally adjacent to `files_in_scope`, even without a direct match, receives a proximity boost. This keeps one formula, one ranked output, and avoids the agent needing to weigh two independently-scored lists against each other.

**Horizontal scaling unit is the repository, not the request.** Queries never need to cross repository or organization boundaries; a repository's Decision Memory and Structure Graph data are self-contained. This makes the natural scaling unit a shard per repository or per organization, not a single ever-growing database that must scale vertically. A routing layer maps `repo_id` to the Postgres instance holding that repository's data (both its Decision Memory tables and its AGE graph, kept together since they are queried together), so horizontal scale is achieved by adding shards as more repositories connect, not by growing one instance without bound.

**Read replicas for the actual load pattern.** Retrieval (`get_relevant_memory`) is the overwhelmingly frequent operation, every agent prompt triggers one. Writes (`resolve_memory`, webhook-driven ingestion, Structure Graph incremental updates) are comparatively rare. Each shard should therefore support read replicas serving retrieval traffic, with writes going to a single primary per shard, rather than provisioning for read and write load as if they were equally frequent.

**Caching for repeated structural traversal.** Call-graph traversal for a commonly-touched module is likely to be queried repeatedly across many agent sessions in a given repository. An in-memory or Redis-backed cache in front of Structure Graph traversal queries, keyed by the traversal query shape and invalidated on the incremental updates described in section 17.5, avoids repeating expensive graph queries that have not changed since the last invalidation.

**What this does not change.** This design does not reopen the packaging decision in section 12 for Stage 1; a single-machine, single-daemon install remains correct for a single team using Zuri locally. This section describes the architecture Zuri grows into once multiple organizations are served from shared infrastructure rather than one daemon per machine, which is the Stage 2 direction already referenced elsewhere in this document, not a revision to what is being built now.

### 17.8 Cross-Service Structural Relationships (Service Boundary Layer)

Sections 17.1 through 17.7 scope Structure Graph per repository, one graph per shard, consistent with the repo-as-shard horizontal scaling design in §17.7. That is correct for a service's internal structure, but real systems, particularly at the scale referenced in this document's roadmap, one company brain eventually understanding hundreds of interdependent services, require representing relationships that cross repository boundaries: Service A calling an API exposed by Service B, Service C consuming an event Service D publishes, and so on. The join-table pattern used for cross-repo Decision Memory in §9.6 does not extend to this problem; that pattern works because integration decisions are sparse and explicitly confirmed one at a time by a human, while structural cross-service edges are dense and cannot realistically be tagged manually at any meaningful scale.

**Design: a separate, deliberately small Service Boundary Layer, not one merged graph.** Each repository's Structure Graph continues to hold its full internal structure exactly as specified in sections 17.1 through 17.6, unchanged. A separate registry holds only each service's public interface, its exposed APIs, published events, and declared external dependencies, along with the edges connecting those public interfaces across services. Because most services expose only a small fraction of their internal structure publicly, this registry remains small relative to the sum of all services' internal graphs even at very large scale; it is not a second copy of everyone's internals, only of the boundaries between them. This mirrors how large organizations already solve this problem in practice, for example via internal service catalogs that track public contracts and ownership without merging every service's internal implementation into one view.

**Populating the boundary layer.** Most services already declare their public interface in a structured, parseable form: OpenAPI specifications, gRPC `.proto` files, AsyncAPI event schemas, and similar contract artifacts. The Structure Graph bootstrap pass (§17.5) should parse these declared contract files directly as part of that pass, not only source code, since they state the public interface explicitly rather than requiring it to be inferred. Where a service calls another without a clean declared contract, for example a raw HTTP call to a hardcoded URL or service name, the Inference Layer (§17.6) resolves which known service that reference actually points to, and writes the resolved edge into the boundary layer, the same division of labor already established between deterministic parsing and AI-driven ambiguity resolution elsewhere in this section.

**Traversal across the boundary.** A structural query that needs to cross a service boundary, for example "what depends on this API," begins within the calling repo's own Structure Graph, crosses into the boundary layer at the point where an internal node maps to a declared or inferred public interface node, and continues into the target service's own Structure Graph on its own shard if deeper internal detail is required there. Every hop that crosses from one repo's shard into another is a genuine network round trip; no design eliminates this cost, and it should not be presented as free.

**Bounding traversal depth by default.** Because unbounded transitive traversal across many services is expensive regardless of how the boundary layer is designed, cross-service structural queries should default to a bounded depth, one or two hops across service boundaries, which answers the large majority of real engineering questions such as "what does this change affect" or "what does this service depend on directly." Full, unbounded, company-wide transitive tracing should be treated as a deliberate, explicitly requested, batch-style operation rather than something every interactive retrieval call pays the cost of. The caching layer specified in §17.7 for repeated structural traversal extends naturally to cache these boundary-crossing paths as well, since a given service-to-service dependency query is likely to be asked repeatedly across many sessions.

**Where this registry lives.** The boundary layer is a cross-cutting concern of the whole installation, not any single repository, and belongs alongside the repo-to-shard routing layer described in §17.7, rather than being owned by any one repo's shard. This keeps the same architectural instinct consistent throughout §17: repo-scoped internals stay local and sharded, cross-cutting concerns get their own deliberately minimal layer rather than being folded into, or requiring, one unified graph spanning everything.
