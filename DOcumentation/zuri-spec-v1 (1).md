# Zuri — Engineering Memory System
## Full Working Spec (v1.0 — Build-Ready)

This document supersedes `zuri-spec.md` (v0.1). Every open question flagged in the v0.1 critical review has been resolved below except where explicitly marked **DEFERRED**. This spec is written to be handed directly to a coding agent (Claude Code, Gemini CLI/Antigravity, or equivalent) as the source of truth for implementation. Sections are numbered for direct reference in commit messages, PR descriptions, and agent prompts (e.g. "implement §10.2").

---

## 1. Purpose

A second brain for a company's engineering organization — good enough that someone who **cannot code** can still direct production-quality changes to the codebase, because the AI coding agent they use is backed by a brain that knows the architecture, the reasoning, the history, and the constraints better than most individual senior developers do.

This is not a search tool for humans. It is an **MCP server** that sits between a coding agent (Claude Code, Gemini CLI, Antigravity, Cursor, etc.) and the company's accumulated engineering memory. On every prompt, the agent queries the brain first, receives relevant context, and only then generates code.

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

  So the invariant becomes **"every canonical record must pass through an explicit review gate,"** rather than "every canonical record originates from a GitHub webhook."

---

## 5. Confidence Threshold

- **Mechanism is fixed, values are configurable.** Every org gets: status (`proposed`/`confirmed`/`lapsed`) + named approver + expiry window.
- Default out of the box (small teams): one named approver, 30–90 day expiry.
- Configurable per org: larger orgs may want multiple approvers / formal RFC sign-off before a probabilistic decision is trusted enough to shape generated code. A small startup may want just one tech lead's approval.
- The system does not hardcode organizational hierarchy — only the state machine. Approver identity is stored as a config value (GitHub username/team), not derived from any assumed org chart.
- **Implementation note**: this config lives in a per-repo `zuri_config` table (see §7.2), editable from the Electron GUI Settings pane for that project — not a flat file, so it survives GUI-driven edits and displays in the "container inspector" UI described in §12.

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

### 7.2 Full Schema (Postgres + pgvector — see §11 for why no separate graph DB)

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

**`zuri_config`** (per-repo, editable via GUI — see §5)
| Field | Type | Notes |
|---|---|---|
| `repo_id` | FK → `repo` | |
| `approver_usernames` | text[] | GitHub usernames authorized to confirm probabilistic memory |
| `expiry_days` | int | default 60 |
| `reminder_cadence_days` | int | default 7 — see §10.4 floor rule |
| `additional_notify_channels` | jsonb | optional Slack webhook etc. — post-v1 stub field, unused in v1 |

**`memory_record`** (shared table across all tiers)
| Field | Type | Notes |
|---|---|---|
| `memory_id` | UUID (PK) | synthetic, immutable identity |
| `repo_id` | FK → `repo` | not a literal repo name/URL |
| `tier` | enum | `canonical` \| `probabilistic` \| `working` |
| `status` | enum | `proposed` \| `confirmed` \| `rejected` \| `lapsed` (canonical is implicitly `confirmed`) |
| `source_type` | enum | `pr_merge` \| `onboarding_survey` \| `agent_session`, defaults to `pr_merge`. Distinguishes how a record entered the system; see §10.5 for the onboarding path. Does not affect tier or status logic, purely descriptive and used for GUI display and filtering. |
| `decision` | text | the inferred/confirmed decision, one to two sentences |
| `reasoning` | text | supporting context — why, not just what |
| `content_embedding` | vector(1536) | pgvector column, generated from `decision` + `reasoning` at write time |
| `originating_commit` | text | commit hash at time of creation — stable, unlike branch name |
| `originating_pr_number` | int | nullable — null for pure working memory not yet in a PR |
| `created_by` | text | GitHub username or agent-session identity |
| `resolved_by` | text | nullable — who confirmed/rejected/edited |
| `branch_label` | text | mutable, human-readable — nullable for canonical (branch may be deleted) |
| `decision_title` | text | mutable label for probabilistic memory — a decision can span multiple branches/PRs over time |
| `created_at` | timestamptz | |
| `resolved_at` | timestamptz | nullable |
| `expires_at` | timestamptz | nullable — only set for `probabilistic` + `proposed` |
| `citation_count` | int | denormalized counter, updated by trigger on `memory_citation` insert |
| `last_cited_at` | timestamptz | denormalized, same trigger |

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

**`audit_log`** (append-only — see §14)
| Field | Type | Notes |
|---|---|---|
| `log_id` | UUID (PK) | |
| `memory_id` | FK → `memory_record`, nullable | null for pure read events with no single memory match |
| `event_type` | enum | `retrieved` \| `confirmed` \| `rejected` \| `edited` \| `lapsed` \| `revival_flagged` |
| `actor` | text | GitHub username, agent-session ID, or `system` |
| `context` | jsonb | free-form — e.g. prompt text hash, PR number, source_context string |
| `occurred_at` | timestamptz | |

### 7.3 Keying by Tier
- **Canonical memory** → keyed by `memory_id`, no meaningful branch label (branch is likely deleted by the time it's canonical).
- **Working memory** → keyed by `memory_id`, indexed via `created_by` + `branch_label` for lookup, but never addressed *by* that label — the label can change/disappear without orphaning the record.
- **Probabilistic memory** → keyed by `memory_id`, indexed via `decision_title`, since a single decision can span multiple branches/PRs over time rather than living on one.

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

### 9.3 Scoring Model

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

### 9.4 Learned Weights (Phase 2 — not v1 blocking)

Rather than hand-setting how much semantic-match should count vs. graph centrality vs. recency, track outcomes: when the agent surfaced memory X and the resulting code merged clean vs. got rejected/reverted in review, that's a training signal for learning-to-rank. **This requires a merge-outcome feedback loop that doesn't exist until the ingestion pipeline (§10) has been live for some weeks** — treat §9.3's fixed formula as the v1 default, and §9.4 as an explicit Phase 2 item once there's outcome data to learn from.

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

### 10.2 Extraction Method

Hardcoded extraction rules are rejected — PR hygiene is inconsistent (sparse descriptions, "fix bug"-style messages, real reasoning scattered across diff/commits/comments/linked issues), so no fixed parser can reliably pull out "the decision." Instead: **an AI model infers** the candidate decision from the full PR context (diff + description + commit messages + review comments + linked issues), the same way a thoughtful reviewer would piece it together manually.

Inference is never trusted silently — it surfaces as a **bot comment directly on the PR itself**, next to the code the human is already reviewing: *"Zuri inferred this decision from this PR: [X]. Confirm, edit, or reject?"* Reviewer replies map to the `resolve_memory` tool (§13.2) via a GitHub webhook listener on comment events. This keeps memory-writing on the exact same gate as code review (§4) — no separate approval workflow for reviewers to skip or forget.

**Merge state signaling**: when the merge webhook handler calls `resolve_memory` internally (either directly or via the bot-comment-reply path), it must pass the `pr_merged` field explicitly per §13.3's schema. `pull_request.closed` with `merged:true` maps to `pr_merged: true`; every other path maps to `pr_merged: false` or omits it. This must be a structural field on the call, never inferred by string-matching `source_context` or any other free-text value.

**Model choice for extraction**: this runs server-side inside the Go daemon, calling out to an LLM API (model-agnostic — configurable per install, default to a fast/cheap model since this is a background inference task, not a user-facing chat). This is a separate concern from which agent/model the *developer* is using in their IDE.

### 10.3 Pipeline Mechanics

- **Read-only GitHub App** (not a personal access token, not full OAuth) — scoped to `pull_requests:read`, `contents:read`, `issues:read`, `metadata:read`. No write scope for repo content in v1: Zuri cannot modify the codebase, only read it. (Zuri *does* write PR comments, which requires `pull_requests:write` narrowly for the comment API only — this is a materially smaller trust ask than modify-code access, and should be called out explicitly as such during GitHub App manifest registration and in any user-facing permissions explanation.)
- **Webhook-driven, not polling** — GitHub pushes events (`pull_request.opened`, `pull_request.synchronize`, `pull_request.closed` with `merged:true`, `pull_request_review_comment.created`, `issue_comment.created` for bot-comment replies) the moment they happen.
- **Webhook receiver lives inside the same Go daemon** as the MCP server — one process, one route (e.g. `POST /webhooks/github`), verified via GitHub's webhook signature (HMAC secret configured at App registration).

```
GitHub webhook fires (PR opened/updated/merged/commented)
      ↓
Zuri fetches PR diff + description + comments + linked issues (read-only)
      ↓
Extraction model infers candidate decision(s)
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

### 10.5 Greenfield Onboarding Ingestion

Sections 10.1 through 10.4 assume PR history exists to extract from. A company on day one has none. The only source of truth at that point is what the founder or founding team can articulate directly, so this requires a distinct ingestion path, not a variant of the PR-based pipeline.

**Mechanism**: an adaptive intake flow, run from the Electron GUI's "Create Brain" wizard, not through the MCP tool surface, since this is a founder-facing setup action, not an agent-facing runtime action, and keeping the MCP tool surface agent-only (per section 13) avoids conflating two different kinds of caller. Each answer determines the next question rather than presenting a fixed form, covering domain, expected scale, which tradeoffs the founder actually cares about, hard constraints such as compliance or data residency, and explicit non-goals. The exact question tree is an implementation detail to design when this phase is opened, not specified here.

**Write path and the gate it substitutes for**: section 4 requires every write to canonical memory to pass through a review gate equivalent to code review, no exceptions. There is no PR to attach this to, so the review moment is the founder explicitly confirming the survey's synthesized output before it is written, screen by screen or as a final summary confirmation, functioning as the same kind of deliberate approval a PR merge represents, just without a diff attached to it.

**Schema change required**: add a `source_type` field to `memory_record` (`pr_merge` | `onboarding_survey` | `agent_session`), defaulting to `pr_merge` for backward compatibility with existing rows from sections 1 through 16. This keeps onboarding-derived memory distinguishable from PR-derived memory in both the database and the GUI. The Electron GUI's record inspector (section 12) should render onboarding-sourced records with a visibly distinct marker, for example a "founding intent" tag alongside the existing tier badge, since a founder's stated intention and a team's proven-in-production decision are different weights of truth even when both carry `tier=canonical`.

**No special-cased decay or expiry**: onboarding-derived memory participates in the same scoring model as any other canonical record (section 9.3), with no separate expiry logic. As real PR-derived canonical memory accumulates on the same topic and gets cited, it naturally outranks stale founding assumptions through the existing trend and recency mechanics, no new code path is needed for founding memory to fade in relevance as the company's real decisions accumulate around and eventually past it.

### 10.6 Backfill Prioritization for Existing Codebases

Running the full extraction pipeline (section 10.2) against years of historical PRs is expensive in LLM tokens and, at real scale, not worth doing exhaustively. Uniform backfill is the wrong default; prioritized backfill against an explicit budget is correct regardless of how much processing time or cost is available, because processing low-impact history at the same priority as high-impact history is not a more thorough approach, it is a worse one.

**Prioritization mechanism**: this depends on Structure Graph (section 17) to be fully realized, since centrality data is what makes prioritization meaningful rather than arbitrary. Once Structure Graph exists: run its deterministic, non-LLM bootstrap pass (section 17.5) first, this produces a call and dependency graph with no inference cost. Use centrality within that graph, files and modules with the most incoming structural connections, to rank historical merged PRs by how many high-centrality files they touched. Feed the extraction pipeline (section 10.2) that ranked list, highest-centrality first, recency as a secondary sort within equal centrality, rather than chronological or arbitrary order.

**Until Structure Graph exists**: backfill prioritization falls back to recency-weighted order alone, most recent merged PRs first, since centrality data is not yet available to rank by structural importance. This is an explicit interim behavior, not a placeholder to silently forget about once section 17 is built.

**Budget, not completeness, is the design target**: the org sets an explicit backfill budget in `zuri_config`, either a token cap or a PR count cap. Processing stops when the budget is exhausted, not when history runs out. This means backfill is deliberately incomplete by design for any codebase with enough history to exceed the budget, the long tail of low-impact historical PRs may never be processed, and that is a stated tradeoff, not a defect. Every skipped PR should be logged (which PR, why it was skipped, that it was a budget cutoff rather than an error) so the gap is visible and inspectable, never silently absent.

**An unconfirmed AI-inferred decision must never be silently discarded, and it must never expire quietly.** If a reviewer merges a PR without engaging the bot comment, the underlying code change still lands — memory must not fall behind it.

Rule: an unconfirmed extraction tied to a **merged** PR is written immediately as `tier=probabilistic, status=proposed` (never discarded, never silently promoted to canonical either) and is then **actively re-surfaced, not passively left to expire** — it keeps getting pinned back to the responsible reviewer/team until it is explicitly confirmed or explicitly rejected. This is distinct from ordinary lapsing in §6: ordinary probabilistic memory expires because its proposal window closed with no resolution either way and quietly downgrades; a merged-but-unconfirmed decision keeps actively demanding resolution because the code is already live and the brain's gap against it is a standing risk, not a background one.

**Re-surfacing cadence — hybrid, fixed floor + configurable layer:**

- **Fixed default (always on, not disable-able below a floor):** a time-based reminder cadence ships out of the box (default 7 days, per `zuri_config.reminder_cadence_days`), so no org can silently let a merged-but-unconfirmed decision go unaddressed indefinitely.
- **Event-triggered (always on):** the moment another PR touches the same files/symbols/decision area (checked via `memory_touches_file` overlap), the unconfirmed record is re-flagged immediately, independent of the time-based cadence.
- **Org-configurable layer on top:** the org can tune the time-based interval and add extra notify channels (`zuri_config.additional_notify_channels` — Slack webhook etc., stubbed field in v1 schema, not implemented in v1 logic). What can't be configured away is the floor itself — reminders can be more frequent or routed differently, never disabled outright.

---

## 11. Storage Architecture — Postgres + pgvector Only (No Graph DB)

**Decision, closed**: a single Postgres database with the pgvector extension is the entire storage layer. No Neo4j, no Apache AGE, no second database of any kind in v1.

**Why**: the only traversal need identified — "what files does this memory touch, what other memories touch those same files" — is 1–2 hop traversal, not deep multi-hop graph querying. This is handled by the `memory_touches_file` join table (§7.2) with plain joins or recursive CTEs if traversal depth needs increase later. PageRank-style centrality (§9.3) is computed as a scheduled batch job over `memory_citation`, which is an adjacency structure Postgres represents natively — a dedicated graph engine is not required to compute centrality scores.

**Rejected alternative and why**: introducing Neo4j means a second database to deploy, back up, and keep in sync (dual-write or event-driven sync, each with its own failure modes), and — critically — it breaks the single-binary packaging goal (§12), since Neo4j is a JVM server process that cannot be compiled into one executable. Bundling a JVM inside the Electron installer, or depending on a hosted Neo4j Aura instance, both contradict the "download one EXE, run it, done" distribution model this project is built around. Revisit only if query complexity or traversal depth becomes a measured bottleneck post-launch — not before, and not speculatively.

**Embedded vs. server Postgres for distribution**: for v1, ship an embedded/managed Postgres instance (e.g. via a bundled Postgres binary the Go daemon spawns and manages, or SQLite with a migration path to Postgres if embedding proves too heavy for the installer size target — **this specific sub-choice is left to implementation discretion**, since it doesn't affect the schema or any decision above, only how the daemon starts up).

---

## 12. Packaging & Distribution

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
- Per-record inspector: click a single memory record to see full history — decision text, reasoning, source PR, citation count/trend, resolution history. This is the "`docker inspect`" equivalent.
- Settings pane per project: edit `zuri_config` values (approvers, expiry window, reminder cadence) — this is the GUI surface for §5's configurability.
- Live activity feed: real-time stream (via the SSE endpoint, §13.3) showing "Agent X just queried memory for prompt Y" as it happens.

**GUI is a management/inspection surface only — it is not the memory itself and not the primary product interface.** The MCP server + agent integration is the primary product; the GUI exists so a human can see what the daemon is doing, same as Docker Desktop exists to inspect a system that runs fine headless.

---

## 13. Transport & MCP Tool Contracts

### 13.1 Transport Decision

MCP defines two official transports — stdio (host process spawns the server directly, pipes over stdin/stdout) and Streamable HTTP (server runs as a long-lived process on a local port; can use SSE for server-initiated push, not for streaming tool-call responses token-by-token). Both are valid ways to build an MCP server; the tool-handling logic is transport-agnostic in the standard SDKs. Implementation uses the official Go SDK (`github.com/modelcontextprotocol/go-sdk`), which supports both transports and is spec-complete, so switching from the originally considered TypeScript SDK loses no transport capability.

**Decision: Streamable HTTP for the MCP server itself.** Reason: the daemon is long-running and GUI-managed rather than spawned per-agent-session, and multiple agents (e.g. Claude Code and Gemini CLI open simultaneously on the same machine) need to hit **one running daemon instance** rather than each spawning their own server process.

**Decision: a separate SSE endpoint, not on the MCP tool-call path, purely for the GUI's live activity feed** (§12). This is a legitimate server-push use case — pushing "agent X just queried Y" events to the Electron UI in real time — distinct from and not to be confused with the MCP tool-call request/response, which is single-shot: the retrieval computation (Stage 1 + Stage 2, §9) completes server-side before a response is returned, so there is no meaningful partial result to stream mid-computation on that path.

**Testing Requirement**: An explicit end-to-end test requirement MUST exercise the actual Streamable HTTP server by POSTing/requesting through the registered `http.ServeMux` rather than invoking handlers directly. This prevents HTTP routing regressions (such as route handler shadowing/overrides) that unit tests against handler functions alone would miss, complementing the transport requirements without altering the underlying architecture.

### 13.2 Tool: `get_relevant_memory`

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
- `confirm` with `pr_merged=true` → `tier` → `canonical`, `status` → `confirmed`, permanent, `expires_at` cleared.
- `confirm` with `pr_merged=false` or omitted → stays `tier=probabilistic`, `status=confirmed`, `expires_at` set per `zuri_config.expiry_days` from `resolved_at`.
- `reject` → `status=rejected`, record retained (not deleted), excluded from active ranking (`status_weight=0` per §9.3).
- `edit` → `decision` field is overwritten with `edited_content` before the `confirm` logic above applies — this is the mechanism that prevents a bad AI inference from calcifying into canonical truth.
- This tool **can only transition an existing `proposed` record** — it has no code path to create a new `confirmed` canonical record directly, mechanically enforcing §4.
- Every call writes an `audit_log` entry with `event_type` matching the action taken (`confirmed`/`rejected`/`edited`), regardless of outcome.

---

## 14. Audit Logging & Access Model

- **v1 scope: audit logging is fully implemented; role-based access control (RBAC) enforcement is DEFERRED.**
- Every read (`get_relevant_memory`) and every write (`resolve_memory`, ingestion pipeline events, lapse events, revival flags) writes an append-only row to `audit_log` (§7.2). This alone gives full traceability: who/what queried which memory, when a memory was created/confirmed/rejected/edited and by whom.
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

---

## 16. Build Order (Suggested)

This is a suggested sequence for a coding agent to follow, front-loading the pieces other pieces depend on:

1. **Postgres schema** (§7.2) — all tables, indexes, the `memory_citation`-driven trigger for `citation_count`/`last_cited_at`.
2. **Go daemon skeleton** — process that boots, manages embedded Postgres, exposes health check.
3. **MCP server, Streamable HTTP transport** (§13.1) — wire up `get_relevant_memory` and `resolve_memory` (§13.2/13.3) against the schema, even with stubbed/naive scoring initially. Must include end-to-end HTTP integration tests that request through the registered `http.ServeMux` (rather than isolated handler function calls) to ensure routing correctness and prevent route handler shadowing.
4. **Scoring model** (§9.3) — replace naive scoring with the real formula once retrieval plumbing works end-to-end.
5. **GitHub App registration + webhook receiver** (§10.3) — manifest, signature verification, event routing.
6. **Extraction pipeline** (§10.2) — LLM inference call + bot comment posting.
7. **Bot-comment-reply → `resolve_memory` parsing** (§10.3 diagram) — closes the ingestion loop.
8. **Persistent pending / re-surfacing logic** (§10.4) — scheduled job + event-triggered re-flagging.
9. **Electron GUI shell** (§12) — project list, per-record inspector, settings pane, live activity feed (wire to SSE endpoint, §13.1).
10. **Auto MCP-config registration on first auth** (§8).
11. **Audit log surfacing in GUI** (§14) — not required for daemon correctness, but needed before this is usable by a non-technical stakeholder per §1's premise.

## 17. Structure Graph & Inference Layer (Deferred Phase — Not Current Build Scope)

Everything in sections 1 through 16 describes one of three subsystems Zuri is ultimately composed of. Only the first is built. The other two are named and scoped here so they are not lost, and so nothing in the current build accidentally forecloses them, but no part of this section is in scope until explicitly reopened as a new phase.

### 17.1 The Three Subsystems

- **Decision Memory** (built, S1 through S3): the canonical/probabilistic/working tiers described in sections 3 through 14. Postgres plus pgvector. Answers "why is this the way it is." PR-gated per section 4.
- **Structure Graph** (deferred): a language-agnostic structural map of the codebase itself. Answers "what exists and how is it connected," not why. Deterministic, not inferred from prose the way Decision Memory extraction is.
- **Inference Layer** (deferred): the AI-driven subsystem that resolves structural ambiguity the parser alone cannot, and places that resolved meaning onto the Structure Graph. Distinct from the extraction pipeline in section 10.2, which infers decisions from PR text for Decision Memory. The Inference Layer infers structural meaning from code itself, for example recognizing that a given module functions as the actual payment boundary even though no single line of code states that.

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
