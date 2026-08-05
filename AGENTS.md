# AGENTS.md — Engineering Directives & Code Quality Standards

This document establishes the binding engineering standards, architectural invariants, and verification protocols for any AI Agent or human engineer contributing to the **ZURI** codebase.

---

## 1. Core Philosophy & Quality Directives

1. **Production-Grade Exclusively**:
   * Optimize exclusively for correctness, architecture, maintainability, scalability, readability, and long-term evolution.
   * Ignore implementation speed, token budget, developer convenience, or the quantity of code required. Never select a shortcut simply because it is faster to write.
   * Assume all code will undergo rigorous review by principal engineers from Go Core, Google, Meta, Uber, Stripe, Cloudflare, and CockroachDB. Zero technical debt tolerance.

2. **Silent Engineering Process**:
   Before outputting or committing any code, every implementation must silently follow this exact step-by-step discipline:
   * **Problem Comprehension**: Fully understand domain requirements and system constraints.
   * **Invariant Identification**: Define preconditions, postconditions, and structural invariants.
   * **Edge Case Enumeration**: Identify boundary conditions, null/empty states, timeout cases, and race conditions.
   * **Architectural Design**: Establish package boundaries, domain interfaces, and DTO mappings prior to writing code.
   * **Failure Mode Analysis**: Identify partial failures, network partitions, database deadlocks, and cancellation paths.
   * **Implementation**: Write explicit, clean, un-clever code.
   * **Rigorous Self-Review**: Perform a self-PR review checking correctness, race safety, memory allocation, error context, and testability. Refactor anything below production standards before finalizing.

---

## 2. Go Language & Idiomatic Design Standards

* **Idiomatic Go**: Adhere strictly to *Effective Go* and standard library design principles.
* **Interface Design**: Interfaces must be defined by the consumer, not the producer. Never introduce unnecessary indirection or speculative interfaces.
* **Composition over Inheritance**: Use embedding only when structural composition represents a true IS-A relationship. Prefer explicit fields and delegation.
* **Package Cohesion & Isolation**:
  * Packages must represent single cohesive domains (e.g., `pkg/scoring`, `pkg/gaps`, `pkg/db`).
  * Business logic must remain independent from transport (`HTTP`, `SSE`, `CLI`, `TUI`) and infrastructure (`PostgreSQL`, `pgvector`).
  * Never allow circular package imports or hidden package-level coupling.
* **State & Scope**:
  * Zero global mutable state. All state must be encapsulated within explicitly instantiated struct instances passed via dependency injection.
  * Avoid magic values, hardcoded strings, or arbitrary integer offsets. Define explicit constants and enums.

---

## 3. Concurrency, Cancellation & Resource Management

* **Correctness First**: In concurrent code, safety and correctness strictly supersede throughput optimizations.
* **Context Propagation**: Every blocking call, network request, DB transaction, and background goroutine MUST accept and respect `context.Context`.
* **Goroutine Lifecycle**:
  * Every goroutine spawned must have a deterministic termination path tied to context cancellation or a closed channel.
  * Never leak goroutines. Always use `sync.WaitGroup` or managed worker channels to ensure clean shutdown during daemon termination.
* **Race Condition Safety**: Run and pass all tests with `go test -race ./...`. No shared state mutations without explicit mutex protection or channel synchronization.

---

## 4. Error Handling & API Ergonomics

* **No Silenced Errors**: Never ignore, swallow, or discard errors (`_ = err` is prohibited unless explicitly justified in a comment for non-failable standard library functions like `w.Write` in HTTP handlers where header is written).
* **Contextual Error Propagation**: Always wrap lower-level errors with operational context:
  ```go
  if err != nil {
      return fmt.Errorf("evaluating evidence strength for memory %s: %w", recordID, err)
  }
  ```
* **Misuse-Resistant APIs**: Constructors (`NewService(...)`) must validate input dependencies and return explicit error states when invalid.

---

## 5. Clean Code & Documentation Rules

* **Self-Documenting Code**: Code must be readable without requiring comments to explain *what* the code does. Comments exist exclusively to explain *why* non-obvious design choices or domain rules exist.
* **No Placeholders or TODOs**: Never commit `TODO`, `FIXME`, dummy fallbacks, empty handlers, or unhandled branches.
* **Exported Symbols**: Every exported type, function, struct, and constant must have a clear, godoc-compliant comment explaining its responsibility.

---

## 6. Verification Checklist

Before declaring any feature complete or merging a pull request:
- [ ] `go test -v -race ./...` passes cleanly with zero race warnings.
- [ ] Package boundaries are respected (no domain logic leaked into transport/CLI packages).
- [ ] Context cancellation and timeout propagation are verified.
- [ ] Audit logs and execution history are properly updated in [`documentation/zuri-build-execution-log.md`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/documentation/zuri-build-execution-log.md).
