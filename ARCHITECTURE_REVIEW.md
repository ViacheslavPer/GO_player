# Architecture Review Summary - MVP v2.0

## Review Date: 2026-02-12

## Architecture Overview

The project follows a clean separation of concerns with the following components:

### 1. BaseGraph (`internal/memory/basegraph/`)
- **Role**: Long-term memory storage
- **Characteristics**: 
  - Pure data structure with integer weights (stored as float64)
  - No probabilities, no runtime logic, no UX logic
  - Thread-safe operations (no mutexes needed - single writer pattern expected)
- **Operations**: Reinforce, Penalty, GetEdges, GetAllIDs
- **Status**: ✅ Clean separation, no dependencies on other modules

### 2. RuntimeGraph (`internal/memory/runtime/`)
- **Role**: Runtime weights + cooldowns + penalties
- **Characteristics**:
  - Thread-safe with mutexes (edMu, cdMu, pMu)
  - Computes probabilities from base weights + cooldowns + penalties
  - Tracks build version, timestamp, and diffts counter
- **Operations**: CopyBase, Reinforce, Penalty, AddCooldown, ReduceCooldown, GetEdges
- **Dependencies**: Depends on BaseGraph (for CopyBase)
- **Status**: ✅ Clean separation, does not depend on Selector

### 3. Selector (`internal/memory/selector/`)
- **Role**: Pure selection policy
- **Characteristics**:
  - Uses Gini coefficient to decide between TopK and weighted selection
  - Does NOT mutate RuntimeGraph (verified by existing tests)
  - Read-only access to RuntimeGraph
- **Operations**: Next (selects next song ID based on probabilities)
- **Dependencies**: Depends on RuntimeGraph (read-only)
- **Status**: ✅ Pure function, no side effects

### 4. PlaybackChain (`internal/playback/`)
- **Role**: Navigation history only
- **Characteristics**:
  - Manages BackStack, Current, ForwardStack
  - Tracks LearningFrozen state
  - No business logic, pure navigation state
- **Operations**: Next, Back, Forward, FreezeLearning, UnfreezeLearning
- **Dependencies**: None
- **Status**: ✅ Pure navigation state, no business logic

### 5. Orchestrator (`internal/orchestrator/`)
- **Role**: Coordination layer
- **Characteristics**:
  - Integrates BaseGraph, RuntimeGraph, Selector, PlaybackChain
  - Manages background goroutines for cooldowns and runtime graph rebuilds
  - Coordinates learning (Reinforce) and navigation (PlayNext, PlayBack)
- **Operations**: Learn, PlayNext, PlayBack, RebuildRuntime, Start, Stop
- **Dependencies**: All other modules
- **Status**: ✅ Proper integration layer

## Architecture Consistency Check

### ✅ No Circular Dependencies
- BaseGraph → (no dependencies)
- RuntimeGraph → BaseGraph (one-way)
- Selector → RuntimeGraph (one-way, read-only)
- PlaybackChain → (no dependencies)
- Orchestrator → BaseGraph, RuntimeGraph, Selector, PlaybackChain (one-way)

### ✅ No Business Logic in PlaybackChain
- PlaybackChain only manages navigation state
- LearningFrozen flag is managed but logic is in Orchestrator

### ✅ Selector Does Not Mutate RuntimeGraph
- Verified by existing test: `TestSelector_DoesNotMutateRuntimeGraph`
- Selector only reads probabilities via `GetEdges()`

### ✅ RuntimeGraph Does Not Depend on Selector
- RuntimeGraph has no imports or references to Selector

### ✅ Orchestrator is the Only Integration Layer
- All coordination happens in Orchestrator
- Other modules are independent and testable in isolation

## Potential Issues Found

### 1. BaseGraph.Penalty Bug (Line 36)
There appears to be a bug in `BaseGraph.Penalty()`:
- Line 36: `graph.edges[fromID][toID]--` should likely be `graph.edges[0][toID]--`
- This decrements `fromID->toID` twice instead of decrementing both `fromID->toID` and `0->toID`
- **Impact**: Penalty may not correctly reduce global (0->toID) edge weight
- **Recommendation**: Review and fix if this is incorrect behavior

### 2. RuntimeGraph.Reinforce Not Thread-Safe
- `RuntimeGraph.Reinforce()` modifies `edges` without acquiring `edMu` lock
- This could cause race conditions in concurrent scenarios
- **Impact**: Potential data races if Reinforce is called concurrently
- **Recommendation**: Add proper locking or document single-threaded usage

## Test Coverage Status

### ✅ Has Tests
- Selector: Comprehensive test coverage

### ❌ Missing Tests
- BaseGraph: No tests
- RuntimeGraph: No tests  
- PlaybackChain: No tests
- Orchestrator: No tests
- main_test.go: Does not exist

## Conclusion

The architecture is well-designed with clear separation of concerns. The main gaps are:
1. Missing unit tests for most modules
2. Potential thread-safety issue in RuntimeGraph.Reinforce
3. Potential bug in BaseGraph.Penalty

The architecture follows MVP v2.0 design principles correctly.
