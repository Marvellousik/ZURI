# Zuri — Master Execution Plan (Stages 1 through 4)

## Overview & System Philosophy

Zuri is an **evidence-first engineering memory system**. Its core mandate is to ensure an AI coding agent operates with the full architectural context, historical reasoning, conventions, and constraints of an engineering organization.

This Master Execution Plan breaks down all remaining work across **4 Master Stages**, subdivided into concrete **Task Blocks**. Execution follows strict evidence-first principles:
1. **Evidence over certainty**: Confidence is derived from empirical artifacts (merged PRs, citations, code review), not raw model guesses.
2. **Models are replaceable**: LLM intelligence generates hypotheses; Zuri evaluates, stores, and calibrates them.
3. **No un-reviewed canonical writes**: Every canonical memory record must pass an explicit human review gate.
5. **Architecture Decision Record**: All architectural pivots (Electron deferral, relational symbol tables, SQLite/Postgres abstraction, capability-based model registry, context engine) are formally documented in [`documentation/zuri-architecture-decision-record.md`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/documentation/zuri-architecture-decision-record.md).

---

## 🗺️ Master Stage Roadmap

```
Stage 1: Core Engine & Evidence Schema (v1.1 & RFC §7.4)
  ├── Block 1.1: Migration 003 & DB Models (concern, decision_type, boundary, calibration, gap tables)
  ├── Block 1.2: Model Calibration Engine & Raw Confidence Correction
  ├── Block 1.3: Materialized Evidence Strength Engine & Re-calculation Triggers
  └── Block 1.4: MCP Server Extensions (resolve_knowledge_gap & confidence payload block)

Stage 2: Knowledge Recovery & Gap Engine (§10.7)
  ├── Block 2.1: Knowledge Gap Detection Engine (conflicts, unowned, low evidence, stale)
  ├── Block 2.2: Decision Key Gap Clustering & CODEOWNERS Routing
  └── Block 2.3: Batched Gap Digest Engine & Baseline Resolution Seeding

Stage 3: Native Go CLI & Interactive TUI Interface
  ├── Block 3.1: Desktop Electron GUI Deferral & Contract Preservation Spec
  ├── Block 3.2: Native Go CLI Framework (`zuri status`, `query`, `gaps`, `audit`, `mcp`)
  ├── Block 3.3: Interactive Terminal User Interface (`zuri tui` Dashboard)
  └── Block 3.4: Unified Binary Compilation & Local MCP Auto-Registration

Stage 4: Structure Graph & Inference Layer Foundation (§17 - Future Phase)
  ├── Block 4.1: Tree-sitter Universal Code Parser & Universal Schema Nodes
  ├── Block 4.2: Apache AGE Postgres Graph Traversal
  └── Block 4.3: Unified Dual-Subsystem Retrieval & Proximity Boosting
```

---

## 📦 Stage Details & Task Blocks

### STAGE 1: Core Engine & Evidence Schema (v1.1 & RFC §7.4)
*Objective: Upgrade the database schema, models, extraction pipeline, scoring engine, and MCP tool contracts to enforce the RFC §7.4 decision classification taxonomy and dual-confidence model.*

#### 🔹 Task Block 1.1 — Schema Migration 003 & Data Models
* **Tasks**:
  1. Create `migrations/003_schema_v1_1_rfc.sql`.
  2. Alter `memory_record` to add `concern`, `decision_type`, `boundary`, `extraction_confidence_raw`, `extraction_confidence`, `evidence_strength`, `evidence_strength_formula_version`.
  3. Update `decision_key` format rule: `boundary:<boundary>/concern:<concern>/decision_type:<decision_type>`.
  4. Create `model_calibration` table: `(model_id, concern, calibration_curve, sample_size, last_updated_at)`.
  5. Create `knowledge_gap` table: `(gap_id, decision_key, scope, gap_type, candidate_hypotheses, affected_memory_ids, status, routed_to, detected_at, last_surfaced_at, resolved_at, resolved_by)`.
  6. Update `pkg/db/models.go` struct definitions & enums.
* **Verification**: `go test ./pkg/db/...` passes cleanly.

#### 🔹 Task Block 1.2 — Model Calibration & Raw Confidence Correction Engine
* **Tasks**:
  1. Update `pkg/extraction/extractor.go` LLM extraction prompt to require `extraction_confidence_raw` (0–1) and `concern` enum (`reliability`, `security`, `data`, `architecture`, `performance`, `deployment`, `observability`).
  2. Build `pkg/extraction/calibration.go` to look up per-model calibration curves in `model_calibration` keyed by `(model_id, concern)`.
  3. Calculate `extraction_confidence` prior to DB insertion.
  4. Pass through raw confidence when `sample_size` is below threshold and log uncalibrated state to `audit_log`.
* **Verification**: `go test ./pkg/extraction/...` passes cleanly.

#### 🔹 Task Block 1.3 — Materialized Evidence Strength Engine
* **Tasks**:
  1. Build `pkg/scoring/evidence.go` to compute `evidence_strength` from `citation_count`, `status`, `tier`, `last_cited_at`, and `source_type`.
  2. Add automatic DB re-calculation trigger on `memory_citation` inserts and status updates.
  3. Seed onboarding-derived records (`source_type='onboarding_survey'`) with fixed baseline evidence strength.
* **Verification**: `go test ./pkg/scoring/...` passes cleanly.

#### 🔹 Task Block 1.4 — MCP Tool Contracts & Confidence Payload Extensions
* **Tasks**:
  1. Update `get_relevant_memory` tool output schema in `pkg/mcp/tools.go` to return the `confidence` block (`extraction_confidence`, `evidence_strength`, `rationale`).
  2. Implement `resolve_knowledge_gap` tool handler (`pkg/mcp/tools.go`) with `answer` and `acknowledge_unknown` actions.
  3. Add HTTP integration tests exercising `http.ServeMux` for tool contract validation.
* **Verification**: `go test ./pkg/mcp/...` passes cleanly.

---

### STAGE 2: Knowledge Recovery & Gap Engine (§10.7)
*Objective: Build automated gap detection, clustering by decision key, CODEOWNERS routing, digest batching, and resolution mechanics.*

#### 🔹 Task Block 2.1 — Knowledge Gap Detection Triggers
* **Tasks**:
  1. Build `pkg/gaps/detector.go` to run automated detection queries:
     * `conflicting_conventions`: records sharing a `decision_key` with opposing assertions.
     * `insufficient_evidence`: low confidence/evidence extractions expiring without review.
     * `unowned_decision`: decision keys referenced in active scope with no memory record.
     * `stale_unreinforced`: canonical records with declining evidence strength.
  2. Write detection events to `knowledge_gap` table and `audit_log`.
* **Verification**: `go test ./pkg/gaps/...` passes cleanly.

#### 🔹 Task Block 2.2 — Gap Clustering & Ownership Routing
* **Tasks**:
  1. Cluster detected gaps sharing a `decision_key` (`boundary:<boundary>/concern:<concern>/decision_type:<decision_type>`).
  2. Parse repository `CODEOWNERS` files and commit authorship to route gaps to responsible engineers (`knowledge_gap.routed_to`).
* **Verification**: Unit tests for CODEOWNERS parser and clustering logic.

#### 🔹 Task Block 2.3 — Batched Gap Digest & Resolution Seeding
* **Tasks**:
  1. Build `pkg/gaps/digest.go` for batched digest generation on `zuri_config.gap_digest_cadence_days`.
  2. Wire `resolve_knowledge_gap` tool and GUI actions to update gap status (`answered` / `acknowledged_unknown`) and create baseline memory records.
* **Verification**: End-to-end gap resolution tests.

---

### STAGE 3: Native Go CLI & Interactive TUI Interface
*Objective: Build the zero-dependency native Go CLI (`zuri`) and interactive terminal dashboard (`zuri tui`) while preserving Electron GUI contracts in `documentation/zuri-electron-gui-deferral.md`.*

#### 🔹 Task Block 3.1 — Desktop Electron GUI Deferral & Contract Preservation Spec
* **Tasks**:
  1. Document frozen state of `./desktop` in `documentation/zuri-electron-gui-deferral.md`.
  2. Define backwards-compatibility guarantees for REST/SSE/MCP endpoints.
* **Verification**: Spec document created and verified.

#### 🔹 Task Block 3.2 — Native Go CLI Framework (`zuri`)
* **Tasks**:
  1. Build `cmd/zuri/main.go` and subcommand dispatcher package `pkg/cli/`.
  2. Implement `zuri status` (daemon/DB connectivity check).
  3. Implement `zuri query "<prompt>"` (memory retrieval with `--boundary`, `--concern`, `--min-confidence`).
  4. Implement `zuri gaps [list|resolve]` (view open knowledge gaps & submit resolution responses).
  5. Implement `zuri audit` (query operational audit logs).
  6. Implement `zuri daemon [start|stop|status]` (daemon process lifecycle management).
* **Verification**: `go test -race ./pkg/cli/...` passes cleanly.

#### 🔹 Task Block 3.3 — Interactive Terminal User Interface (`zuri tui`)
* **Tasks**:
  1. Build full-screen keyboard-driven terminal dashboard under `pkg/tui/`.
  2. Create Knowledge Gap Inbox view with interactive review & resolution controls.
  3. Create Dual-Confidence Memory Record Inspector view with extraction confidence & evidence strength visualizations.
  4. Create Live SSE Audit Event log view.
* **Verification**: `go test -race ./pkg/tui/...` passes cleanly.

#### 🔹 Task Block 3.4 — Unified Binary Compilation & Local MCP Auto-Registration
* **Tasks**:
  1. Configure `go build` output for `zuri.exe` and `zuri-daemon.exe`.
  2. Add `zuri mcp config` tool to auto-register ZURI server into `.mcp.json`, Claude Desktop, and Gemini CLI.
  3. Update `start-dev.ps1` dev runner script.
* **Verification**: Clean build and end-to-end integration tests.

---

### STAGE 4: Structure Graph & Inference Layer Foundation (§17 - Future Phase)
*Objective: Lay the foundation for universal code structure parsing (tree-sitter), AGE graph storage, and proximity boosting.*

#### 🔹 Task Block 4.1 — Tree-sitter Universal Code Parser & Graph Schema
* **Tasks**:
  1. Integrate `tree-sitter` grammars for Go, TypeScript, Python, etc.
  2. Build parser to map AST nodes into universal schema (Repository, Service, Module, Function, API).

#### 🔹 Task Block 4.2 — Apache AGE Postgres Graph Storage & Traversal
* **Tasks**:
  1. Enable Apache AGE extension in Postgres.
  2. Create Cypher graph schema for call-graphs and service dependencies.

#### 🔹 Task Block 4.3 — Unified Retrieval & Proximity Boosting
* **Tasks**:
  1. Fan-out `get_relevant_memory` to perform structural graph traversal alongside pgvector semantic search.
  2. Fold graph proximity boost into Section 9.3 scoring formula.

---

## 📝 Progress Logging Protocol

Every completed block will be documented in [`documentation/zuri-build-execution-log.md`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/documentation/zuri-build-execution-log.md) with:
- **Timestamp & Stage/Block Identifier**
- **Exact Code Changes & Created Files**
- **Test Commands Run & Empirical Output**
- **Rationale & Architectural Verification**
