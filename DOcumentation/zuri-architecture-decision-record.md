# ADR-002: Architecture Decision Record — Evolution from Electron GUI to Native Go CLI/TUI & Pragmatic V2.1 Architecture

* **Status**: ACCEPTED & ACTIVE
* **Date**: August 5, 2026
* **Authors**: Principal Systems Architect & Engineering Team
* **Scope**: ZURI Core Engine, CLI/TUI Interface, Storage Layer, Model Registry, and Context Synthesis Pipeline

---

## 1. Executive Context & Decision Summary

This Architecture Decision Record (ADR) documents the architectural evolution of ZURI from its initial desktop GUI prototype to the production-grade, terminal-native AI engineering platform specified in **Architecture Blueprint v2.1**.

It details:
1. Why the initial **Electron Desktop GUI** was deferred.
2. The initial **v2.0 Architectural Proposal** (Apache AGE, mandatory embedded PostgreSQL, gRPC plugin runtime).
3. The **Staff Design Review Critiques** identifying overengineering risk and missing core components.
4. The **v2.1 Final Architecture** implementing 5 core architectural pivots.

---

## 2. Chronological Decision Evolution & Pipeline

```
+---------------------------------------------------------------------------------------------------+
| STEP 1: INITIAL STATE (v1.0)                                                                      |
| - Electron + React + Vite Desktop GUI                                                             |
| - Heavy runtime footprint (~350MB+ RAM), separate Node/Vite build pipeline                        |
+---------------------------------------------------------------------------------------------------+
                                                  |
                                                  v  [DEFERRED]
+---------------------------------------------------------------------------------------------------+
| STEP 2: INITIAL PROPOSAL (v2.0 Blueprint)                                                         |
| - Shift to Go CLI + TUI                                                                           |
| - Proposed: Apache AGE graph extension, embedded Postgres everywhere, external gRPC plugins       |
+---------------------------------------------------------------------------------------------------+
                                                  |
                                                  v  [DESIGN REVIEW CRITIQUE]
+---------------------------------------------------------------------------------------------------+
| STEP 3: STAFF ENGINEER REVIEW                                                                     |
| - Overengineering identified (Apache AGE too complex, embedded Postgres shipping risks)           |
| - Missing core components identified (Context Engine, Session Engine, Agent Orchestrator)         |
+---------------------------------------------------------------------------------------------------+
                                                  |
                                                  v  [FINAL REVISION]
+---------------------------------------------------------------------------------------------------+
| STEP 4: FINAL APPROVED ARCHITECTURE (v2.1 Architecture Specification)                             |
| - Terminal-Native CLI & Command Palette TUI (`Space` / `/` key activation)                        |
| - Relational Symbol Tables + SQL Recursive CTEs (Replaced Apache AGE)                             |
| - Pluggable Storage Engine (`pkg/storage`: SQLite + `sqlite-vec` local, Postgres + `pgvector` ent)  |
| - Internal Go Interfaces First (Deferred external gRPC plugins)                                   |
| - Capability-Based Model Registry & Dynamic Model Router                                          |
| - First-Class Context Engine, Session State, and Verification-Gated Agent Orchestrator            |
+---------------------------------------------------------------------------------------------------+
```

---

## 3. Comprehensive Decision Matrix & Rationale

| Architectural Area | Initial Proposed Decision (v2.0) | Reason For Deferral / Modification | Final Approved Decision (v2.1) | Architectural Rationale & Trade-Off |
| :--- | :--- | :--- | :--- | :--- |
| **User Interface** | Electron Desktop App (React/Vite) | High RAM overhead (~350MB+), build pipeline friction, detached from terminal-centric developer workflow. | **Terminal-Native Go CLI & Interactive TUI (`zuri` / `zuri tui`)**. | Single compiled binary (<15MB RAM), sub-10ms startup, keyboard-first interactive command palette (`Space`/`/`), rich panels. |
| **Graph Infrastructure** | Apache AGE Graph Extension | Required maintaining Postgres + pgvector + AGE extension binaries; excessive operational complexity for code relationships. | **Relational Symbol Tables (`symbol_node`, `symbol_edge`) + SQL Recursive CTEs**. | Eliminates graph extension dependency. 2-hop caller/callee traversals executed natively via standard ANSI SQL CTEs. |
| **Database Engine** | Embedded PostgreSQL Everywhere | Shipping and upgrading full PostgreSQL binaries across Windows/macOS/Linux local installs introduces installation & permission friction. | **Pluggable Storage Abstraction (`pkg/storage`)**. Default local: **SQLite + `sqlite-vec`**. Enterprise: **PostgreSQL + `pgvector`**. | Provides zero-config single-file local setup for CLI users while allowing seamless enterprise scaling to Postgres server backend. |
| **Plugin Subsystem** | External gRPC Plugin Runtime Upfront | High complexity (IPC overhead, versioning, security, binary management) before core APIs stabilized. | **Internal Go Interfaces First (`LLMProvider`, `EmbeddingProvider`)**. External gRPC plugins deferred to future phase. | Keeps MVP codebase clean, fast to compile, and easy to refactor. Enables internal compiled-in providers (Ollama, Anthropic, OpenAI, Local ONNX). |
| **Model Registry** | Category-Based Classification (`local_gguf`, `remote_api`) | Static categories failed to express dynamic features (e.g. vision, reasoning, tool calling, context window sizes). | **Capability-Matrix & Dynamic Model Router (`Capabilities` struct)**. | Models advertise capabilities (`ToolCalling`, `Reasoning`, `Vision`, `MaxContextWindow`). Router automatically matches task requirements to best model. |
| **Context Synthesis** | Implicit Context Assembly | Lacked formal pipeline for intent parsing, structural symbol expansion, evidence re-ranking, and token budget packing. | **Dedicated Context Engine (`pkg/contextengine`)**. 9-Stage synthesis pipeline. | Guarantees deterministic, evidence-backed context packing with exact file/line citations and AST proximity boosting. |
| **Task State** | Stateless Prompt-Response Loop | Conversations lost context between commands; lacked active files, reasoning history, and objective tracking. | **Persistent Session Engine (`pkg/session`)**. | Maintains task state, active workspace scope, working files, tool execution logs, and reasoning trajectories across command invocations. |
| **Agent Execution** | Raw Generation Output | LLM output was returned directly without verification or syntax checking, leading to potential unhandled code errors. | **Verification-Gated Agent Orchestration Loop (`pkg/orchestrator`)**. | Enforces a deterministic loop: `Planner -> Retriever -> Tool Executor -> LLM Generator -> Verifier Gate (Syntax / Test Check) -> User Response`. |

---

## 4. Preservation & Migration Contracts

### 4.1 Electron Desktop Codebase (`desktop/`)
* The existing React/Vite/Electron application in `desktop/` is preserved in a frozen state.
* The daemon HTTP/REST and MCP endpoints (`/health`, `/api/repositories`, `/api/memory/query`, `/api/audit-log`, `/mcp`) maintain strict backwards compatibility.
* Refer to [`documentation/zuri-electron-gui-deferral.md`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/documentation/zuri-electron-gui-deferral.md) for full desktop resumption instructions.

### 4.2 Storage & Migration Path
* All schema definitions are maintained in `pkg/storage/migrations/`.
* SQLite schema (`sqlite-vec`) and PostgreSQL schema (`pgvector`) use identical table models (`memory_record`, `knowledge_gap`, `symbol_node`, `symbol_edge`, `audit_log`), enabling 1-to-1 data migration between local and server environments.

---

## 5. Architectural Verification & Compliance

Before declaring any feature complete:
- [x] All Go code must follow [`AGENTS.md`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/AGENTS.md) production standards.
- [x] Interfaces must be defined by consumers in Go domain packages.
- [x] No hidden global mutable state; all state injected via struct constructors.
- [x] All tests run and pass cleanly (`go test ./...`).
- [x] Progress updated in [`documentation/zuri-build-execution-log.md`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/documentation/zuri-build-execution-log.md).
