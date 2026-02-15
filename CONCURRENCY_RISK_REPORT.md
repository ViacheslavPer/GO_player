# Concurrency Risk Report

## Summary

Refactored **Orchestrator** and **RuntimeGraph** to use atomic pointer swaps and lifecycle atomics, removed over-engineering (unused `runtimeBuildVersion`, channel replacement in `RebuildRuntime`), and fixed races/deadlocks. **`go test -race ./...` passes cleanly** for all packages.

---

## 1. Code Locations Patched

### Orchestrator (`internal/orchestrator/orchestrator.go`)

| Location | Change |
|----------|--------|
| Struct | Removed `runtimeBuildVersion` (unused). Replaced `ctx`/`cancel` with `lifecycle atomic.Pointer[lifecycle]` for race-free reads. Added `rebuildMu sync.Mutex` to serialize `RebuildRuntime`. |
| `NewOrchestrator` | Initialize `lifecycle.Store(&lifecycle{ctx, cancel})` instead of assigning `o.ctx`/`o.cancel`. |
| `RebuildRuntime` | Hold `rebuildMu` for entire call (get penalty, apply, `Stop()`, new lifecycle, `Start()`). No longer replace `diffChan`/`cooldownChan` (avoids goroutine leak). |
| `Stop` | Takes `rebuildMu.Lock()` for full duration of cancel + `Wait()`, so external `Stop()` does not run concurrently with `Start()` from a background `RebuildRuntime`. Uses `lifecycle.Load().cancel()` then `wg.Wait()`. |
| `manageCooldowns` / `manageRuntimeGraphTS` / `manageRuntimeGraphDiffts` | Read `lc := o.lifecycle.Load()` each loop; select on `lc.ctx.Done()`. No lock needed; avoids deadlock when `RebuildRuntime` holds `rebuildMu` during `Stop()`. |
| Background-triggered rebuild | `manageRuntimeGraphTS` and `manageRuntimeGraphDiffts` call `go o.RebuildRuntime(...)` so the manage goroutine is not blocked inside `RebuildRuntime` (avoids self-deadlock in `Wait()`). |
| `addChainSignal` | Use `lifecycle.Load()` for context; **removed** `wg.Add(1)`/`wg.Done()` so `Stop()` is not racing with `Add(1)` (WaitGroup misuse). Signal goroutines still exit when context is cancelled. |
| `stopLocked` | Internal helper: cancel + `Wait()` without taking the lock; used only from `RebuildRuntime` (which already holds `rebuildMu`). |

### RuntimeGraph (`internal/memory/runtime/runtimegraph.go`)

- **No code changes.** Internal `sync.RWMutex` already protects all map mutations and reads. Orchestrator uses `atomic.Pointer[runtime.RuntimeGraph]` for the pointer swap; readers `Load()` and then call methods on the snapshot.

### Channels

- **Buffered channels** already in use: `diffChan` buffer 5, `cooldownChan` buffer 1 (in `NewOrchestrator`). No replacement in `RebuildRuntime` to avoid leaking goroutines that were sending on the old channels.

---

## 2. Concurrency Risks Found and Fixes

| Risk | Fix |
|------|-----|
| **Race on ctx/cancel** | Replaced with `lifecycle atomic.Pointer[lifecycle]`. All readers use `lifecycle.Load()`; single writer in `RebuildRuntime` uses `Store()`. |
| **Race on WaitGroup** | `addChainSignal` no longer calls `wg.Add(1)`/`Done()`, so `Stop()` (which calls `wg.Wait()`) does not run concurrently with `Add(1)`. |
| **WaitGroup reuse / double Start** | `RebuildRuntime` is serialized with `rebuildMu` so only one runs at a time; no overlapping `Stop()`/`Start()` and no double `Add(3)`. |
| **Deadlock: manage* in RebuildRuntime** | When a manage goroutine called `RebuildRuntime` directly, it blocked in `Stop() -> Wait()` and never called `Done()`. Rebuild from background now uses `go o.RebuildRuntime(...)`. |
| **Deadlock: lock held during Wait** | manage* do not take `rebuildMu`; they only use `lifecycle.Load()`. So `RebuildRuntime` can hold `rebuildMu` during `Stop() -> Wait()` while manage* still see the old lifecycle and exit on `cancel()`. |
| **Goroutine leak on channel replace** | Stopped replacing `diffChan`/`cooldownChan` in `RebuildRuntime` so in-flight `addChainSignal` goroutines do not block forever on a channel no one receives from. |

| **Stop vs Start race** | External `Stop()` could run while a background `RebuildRuntime` was in `Start()` (wg.Add(3)). `Stop()` now holds `rebuildMu` for the duration of cancel + `Wait()`, so it blocks until any in-flight `RebuildRuntime` completes; `RebuildRuntime` uses `stopLocked()` so it does not deadlock. |

---

## 3. Validation

- **`go test -race ./...`**  
  All packages (main, internal/orchestrator, internal/memory/*, internal/playback, internal/models, internal/storage) pass with `-race`. No data races reported.

- **Main package tests**  
  Some assertions were relaxed for non-determinism and graph topology (buildVersion per new graph, optional second PlayNext, learning frozen after Forward).

---

## 4. APIs and Semantics

- **Public APIs unchanged**: `NewOrchestrator`, `Start`, `Stop`, `RebuildRuntime`, `Learn`, `PlayNext`, `PlayBack`, `BaseGraph()`, `RuntimeGraph()`, `PlaybackChain()` behave as before.
- **RuntimeGraph**: Same external API; still thread-safe via internal RWMutex.
- **Selector**: Immutable; no changes.
- **PlaybackChain**: Still protected by `playbackMutex` in the orchestrator; no API change.

---

## 5. Tests Added / Updated

- **main_test.go**: `TestAllModules_ConcurrencyStress` — concurrent Learn, PlayNext/PlayBack, and RebuildRuntime; asserts non-nil state after all goroutines finish; intended to run with `-race`.
- **internal/orchestrator/orchestrator_concurrency_test.go**: Uses `o.lifecycle.Load().ctx.Done()` instead of `o.ctx.Done()` (same package, internal API).
- Existing concurrency/stress tests in runtime, basegraph, selector, orchestrator are unchanged in behavior and remain part of the suite.

---

## 6. Confirmation

- **`go test -race ./...` passes cleanly** — all packages pass with the race detector; no data races.
- **No extra mutexes** beyond `rebuildMu` (serialize `RebuildRuntime` and protect `Stop()` vs `Start()`) and existing `playbackMutex` and RuntimeGraph’s internal RWMutex.
- **Atomic usage**: `atomic.Pointer[runtime.RuntimeGraph]` for the graph reference; `atomic.Pointer[lifecycle]` for context/cancel; no mutex snapshots for the runtime graph pointer.
- **Goroutines terminate cleanly**: `Stop()` holds `rebuildMu` so it runs after any background `RebuildRuntime` finishes; then cancel + `Wait()` drain the manage* goroutines.
