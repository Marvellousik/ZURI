# Zuri Engineering — AI Anti-Patterns & Post-Mortem Log ("AI Fuck-Ups Not To Repeat")

> **Purpose**: This document records real anti-patterns, developer friction points, and code quality mistakes encountered during Zuri development. It serves as an authoritative guide on engineering standards to enforce and mistakes **never** to repeat.

---

## 🚫 1. Masking Core Architectural Requirements with Fallback Log Messages

### ❌ The Anti-Pattern
When `pgvector` failed to load in embedded PostgreSQL on Windows (because standard vanilla Postgres binary packages omit `vector.dll`), the initial workaround was writing a log message:
```text
[Zuri DB] Note: pgvector extension is not loaded... Operating in dual text/AST search fallback mode.
```
and allowing the daemon to boot without vector similarity search.

### 💥 Why This Is UNACCEPTABLE
- **Architectural Violation**: Vector similarity search ($1536$-dim embeddings with cosine distance `<=>`) is a **non-negotiable core contract** of Zuri's memory system.
- **Lazy Engineering**: Swallowing an error with a fallback log instead of embedding the required native C shared library (`vector.dll` / `vector.so`) shifts the burden to the end user or degrades core system functionality.

### ✅ The Standard To Enforce
- **Zero-Friction Native Bundling**: Always pre-package pre-compiled native binaries (`pkg/db/vendor_extensions/windows_amd64/extracted/lib/vector.dll`) directly inside the daemon directory.
- **Auto-Deployment on Boot**: On database initialization, `EnsureExtensionFiles()` automatically deploys `vector.dll` to `%USERPROFILE%\.zuri\db\binaries\lib\vector.dll` and all `.sql` / `.control` extension manifests to `share/extension/`.
- **Hard Enforcement**: Vector queries (`SELECT ('[1.0, 0.0]'::vector <=> '[1.0, 0.0]'::vector)`) MUST succeed natively out of the box without requiring Docker or manual user C-compiler installation.

---

## 🚫 2. Mocking UI Views with Placeholder Component Stubs

### ❌ The Anti-Pattern
Leaving tabs like **Memory Explorer**, **Decision Log**, and **System Settings** rendering placeholder stubs like `<DeferredViewFeature tabName={activeTab} />` ("Stage 20+ deferred feature") or storing connected repositories in client-side `localStorage`.

### 💥 Why This Is UNACCEPTABLE
- **"Potato UI" Syndrome**: A desktop app with mock tabs or local-storage fallbacks is not an integrated product—it's a prototype mockup.
- **Broken Contract**: The frontend must always communicate with real daemon HTTP/MCP endpoints.

### ✅ The Standard To Enforce
- **100% Unmocked End-to-End Integration**: Every screen in the desktop app MUST consume real backend REST endpoints (`/api/repositories`, `/api/memory/query`, `/api/audit-log`, `/api/agents`).
- **Real Backend Work**: If a UI feature requires a backend endpoint, build out the exact Go handler ([`pkg/api/handlers.go`](file:///C:/Users/Agada%20Bartholomew/Documents/ZURI/pkg/api/handlers.go)), migration (`007_repositories.sql`), and DB query immediately.

---

## 🚫 3. Socket Leaks in Node.js Health Monitoring (`http.get` Stream Retention)

### ❌ The Anti-Pattern
Executing Node.js HTTP health checks without consuming response data:
```ts
http.get('http://127.0.0.1:7331/health', (res) => {
  if (res.statusCode === 200) { ... }
});
```

### 💥 Why This Is UNACCEPTABLE
- In Node.js, if the response stream `res` is not consumed via `res.resume()` or a data listener, the underlying TCP socket descriptor is held in memory, causing health checks to stall, socket leaks, and false "Offline" UI indicators.

### ✅ The Standard To Enforce
- Always call `res.resume()` immediately inside HTTP response callbacks for health probes.

---

## 🚫 4. Process Lifecycle Desynchronization (Orphaned Process Control)

### ❌ The Anti-Pattern
`DaemonProcessManager.stop()` only attempted to kill `this.process`. When `zuri-daemon.exe` was launched externally or pre-existed, `this.process` was `null`, causing `stop()` to do nothing and subsequent `start()` calls to fail due to port `7331`/`5433` collisions.

### 💥 Why This Is UNACCEPTABLE
- Process managers must manage existing processes gracefully, regardless of how they were spawned.

### ✅ The Standard To Enforce
- **Graceful HTTP API Shutdown**: Expose `POST /api/shutdown` on the Go daemon so clients can request clean SIGTERM shutdowns over HTTP.
- **Multi-Tier Process Termination**: If HTTP shutdown times out or process handle is lost, execute OS-level process cleanup (`taskkill /F /IM zuri-daemon.exe` on Windows, `pkill` on Unix) to guarantee ports are freed.

---

## 📜 Summary Checklist Before Declaring "Done"

1. [ ] **No Fallback Log Dodges**: Did you actually fix the root cause, or did you just write a fallback log?
2. [ ] **Zero Mock Data**: Are all UI components consuming live backend REST/MCP APIs?
3. [ ] **Empirical Runtime Proof**: Did you run `go build`, `npm run type-check`, `npm run build`, and test the running process end-to-end?
4. [ ] **Socket & Memory Hygiene**: Are HTTP streams and DB connections closed cleanly?
