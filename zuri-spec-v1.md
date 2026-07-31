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

**No direct writes to canonical memory — ever. No raw INSERT/UPDATE path exists in the codebase for canonical records.** Memory review is gated by code review, exactly like code:

- On-site developers can't merge to main directly → they also can't write to canonical memory directly.
- A PR that changes canonical memory carries the proposed memory-write alongside it.
- Same reviewers, same approval flow, same moment of landing. Memory is never approved faster or slower than the code describing it.
- No parallel path where memory could bypass the gate that code has to go through.
- **Mechanical enforcement**: the only two write paths into `memory_record` are (a) the ingestion pipeline (§10, triggered exclusively by GitHub webhooks) and (b) the `resolve_memory` MCP tool (§13.2), which itself only transitions existing `proposed` records — it cannot create a `confirmed` canonical record from nothing. There is no admin panel, no direct DB write endpoint, no CLI command that bypasses this. This is an implementation requirement, not a policy statement — the agent building this must not add a "manual override" write path under any circumstance, including for testing/seeding (use fixtures/migrations for that instead).

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
| `created_at` | timestamptz | |

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

**Model choice for extraction**: this runs server-side inside the Bun daemon, calling out to an LLM API (model-agnostic — configurable per install, default to a fast/cheap model since this is a background inference task, not a user-facing chat). This is a separate concern from which agent/model the *developer* is using in their IDE.

### 10.3 Pipeline Mechanics

- **Read-only GitHub App** (not a personal access token, not full OAuth) — scoped to `pull_requests:read`, `contents:read`, `issues:read`, `metadata:read`. No write scope for repo content in v1: Zuri cannot modify the codebase, only read it. (Zuri *does* write PR comments, which requires `pull_requests:write` narrowly for the comment API only — this is a materially smaller trust ask than modify-code access, and should be called out explicitly as such during GitHub App manifest registration and in any user-facing permissions explanation.)
- **Webhook-driven, not polling** — GitHub pushes events (`pull_request.opened`, `pull_request.synchronize`, `pull_request.closed` with `merged:true`, `pull_request_review_comment.created`, `issue_comment.created` for bot-comment replies) the moment they happen.
- **Webhook receiver lives inside the same Bun daemon** as the MCP server — one process, one route (e.g. `POST /webhooks/github`), verified via GitHub's webhook signature (HMAC secret configured at App registration).

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

---

## 11. Storage Architecture — Postgres + pgvector Only (No Graph DB)

**Decision, closed**: a single Postgres database with the pgvector extension is the entire storage layer. No Neo4j, no Apache AGE, no second database of any kind in v1.

**Why**: the only traversal need identified — "what files does this memory touch, what other memories touch those same files" — is 1–2 hop traversal, not deep multi-hop graph querying. This is handled by the `memory_touches_file` join table (§7.2) with plain joins or recursive CTEs if traversal depth needs increase later. PageRank-style centrality (§9.3) is computed as a scheduled batch job over `memory_citation`, which is an adjacency structure Postgres represents natively — a dedicated graph engine is not required to compute centrality scores.

**Rejected alternative and why**: introducing Neo4j means a second database to deploy, back up, and keep in sync (dual-write or event-driven sync, each with its own failure modes), and — critically — it breaks the single-binary packaging goal (§12), since Neo4j is a JVM server process that cannot be Bun-compiled into one executable. Bundling a JVM inside the Electron installer, or depending on a hosted Neo4j Aura instance, both contradict the "download one EXE, run it, done" distribution model this project is built around. Revisit only if query complexity or traversal depth becomes a measured bottleneck post-launch — not before, and not speculatively.

**Embedded vs. server Postgres for distribution**: for v1, ship an embedded/managed Postgres instance (e.g. via a bundled Postgres binary the Bun daemon spawns and manages, or SQLite with a migration path to Postgres if embedding proves too heavy for the installer size target — **this specific sub-choice is left to implementation discretion**, since it doesn't affect the schema or any decision above, only how the daemon starts up).

---

## 12. Packaging & Distribution

**Architecture, Docker-Desktop-style**: an Electron GUI shell wraps a Bun-compiled background daemon. Same split as Docker Desktop (Electron GUI) wrapping the Docker daemon.

- **Bun-compiled binary** = the Brain daemon: embedded Postgres/pgvector, GitHub webhook receiver, MCP server (Streamable HTTP transport — §13.1), SSE endpoint for GUI live activity (§13.3), extraction/ranking logic.
- **Electron app**, packaged via `electron-builder` into a single installer per OS (`.exe` / `.dmg` / `.AppImage`) = the GUI shell. Spawns the Bun daemon as a child process on launch, communicates over localhost.

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

MCP defines two official transports — stdio (host process spawns the server directly, pipes over stdin/stdout) and Streamable HTTP (server runs as a long-lived process on a local port; can use SSE for server-initiated push, not for streaming tool-call responses token-by-token). Both are valid ways to build an MCP server; the tool-handling logic is transport-agnostic in the standard SDKs.

**Decision: Streamable HTTP for the MCP server itself.** Reason: the daemon is long-running and GUI-managed rather than spawned per-agent-session, and multiple agents (e.g. Claude Code and Gemini CLI open simultaneously on the same machine) need to hit **one running daemon instance** rather than each spawning their own server process.

**Decision: a separate SSE endpoint, not on the MCP tool-call path, purely for the GUI's live activity feed** (§12). This is a legitimate server-push use case — pushing "agent X just queried Y" events to the Electron UI in real time — distinct from and not to be confused with the MCP tool-call request/response, which is single-shot: the retrieval computation (Stage 1 + Stage 2, §9) completes server-side before a response is returned, so there is no meaningful partial result to stream mid-computation on that path.

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

**Behavior**: executes Stage 1 candidate retrieval + Stage 2 ranking (§9) against the calling repo's memory only (repo identity resolved server-side from the daemon's active project context, not passed by the caller). Truncates `memories[]` to fit `token_budget`. Every call writes an `audit_log` entry with `event_type='retrieved'`.

### 13.3 Tool: `resolve_memory`

Two possible callers: the GitHub bot-comment flow (parsed reply on a PR) and the coding agent directly (a developer tells their agent mid-session to confirm/reject a decision it just surfaced).

**Input schema:**
```json
{
  "memory_id": "string (UUID), required",
  "action": "confirm | reject | edit, required",
  "edited_content": "string — required only when action is 'edit'; replaces 'decision' before confirming",
  "resolved_by": "string — GitHub username or agent-session identity, required",
  "source_context": "string, optional — e.g. 'PR #601 comment' or 'agent session: <prompt excerpt>', for audit trail"
}
```

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
- `confirm` on a record whose `originating_pr_number` is merged to the default branch → `tier` → `canonical`, `status` → `confirmed`, permanent, `expires_at` cleared.
- `confirm` on a record whose PR is still open/RFC-style → stays `tier=probabilistic`, `status=confirmed`, `expires_at` set per `zuri_config.expiry_days` from `resolved_at`.
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
- Neo4j/graph database of any kind — see §11, closed decision.
- Learned/adaptive ranking weights (§9.4) — v1 ships the fixed formula in §9.3; learning-to-rank is Phase 2, contingent on outcome data existing.
- In-app RBAC — see §14, deferred to multi-seat phase.
- Streaming/SSE on the MCP tool-call path — see §13.1; SSE is used only for the GUI activity feed, not tool responses.

---

## 16. Build Order (Suggested)

This is a suggested sequence for a coding agent to follow, front-loading the pieces other pieces depend on:

1. **Postgres schema** (§7.2) — all tables, indexes, the `memory_citation`-driven trigger for `citation_count`/`last_cited_at`.
2. **Bun daemon skeleton** — process that boots, manages embedded Postgres, exposes health check.
3. **MCP server, Streamable HTTP transport** (§13.1) — wire up `get_relevant_memory` and `resolve_memory` (§13.2/13.3) against the schema, even with stubbed/naive scoring initially.
4. **Scoring model** (§9.3) — replace naive scoring with the real formula once retrieval plumbing works end-to-end.
5. **GitHub App registration + webhook receiver** (§10.3) — manifest, signature verification, event routing.
6. **Extraction pipeline** (§10.2) — LLM inference call + bot comment posting.
7. **Bot-comment-reply → `resolve_memory` parsing** (§10.3 diagram) — closes the ingestion loop.
8. **Persistent pending / re-surfacing logic** (§10.4) — scheduled job + event-triggered re-flagging.
9. **Electron GUI shell** (§12) — project list, per-record inspector, settings pane, live activity feed (wire to SSE endpoint, §13.1).
10. **Auto MCP-config registration on first auth** (§8).
11. **Audit log surfacing in GUI** (§14) — not required for daemon correctness, but needed before this is usable by a non-technical stakeholder per §1's premise.

Everything above is intentionally sequenced so that steps 1–4 give you a working, testable brain before any GitHub/Electron complexity is introduced.
