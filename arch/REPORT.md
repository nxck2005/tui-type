# tui-type architecture report

Date: 2026-07-19

## Executive assessment

`tui-type` has a strong architecture for its current scope: it is small,
acyclic, easy to understand, and unusually testable for a TUI. The pure typing
engine is the main architectural asset and should remain the center of the
design.

The application is not yet ready for substantial mode expansion. Before adding
languages, punctuation, quotes, or word-count tests, it needs a persistent
test-configuration identity. Otherwise results and personal bests from
materially different tests will be mixed together.

The most important current correctness issue is at the test deadline:
keystrokes can still be accepted after time expires but before the next 100 ms
application tick notices the expiry.

## Current architecture

```text
main
 ├─ stats: open local data
 └─ app: Bubble Tea orchestration
      ├─ test: engine and metrics
      ├─ words: embedded English word source
      ├─ stats: persistence and aggregation
      └─ ui: screen rendering
           ├─ test
           └─ stats
```

The package graph is acyclic. The `ui` package is not entirely independent:
it consumes concrete engine and statistics types.

| Package | Present responsibility | Assessment |
|---|---|---|
| `main` | Resolve the store, start Bubble Tea, report fatal startup errors | Correctly minimal |
| `internal/app` | Navigation, input routing, ticking, cursor state, persistence coordination | Appropriate now; likely growth point |
| `internal/test` | Word state, timing, keystrokes, Monkeytype metrics | Best-defined boundary |
| `internal/words` | Embedded English-200 list and random batches | Simple and replaceable |
| `internal/stats` | JSON storage, totals, PBs, and profile aggregates | Suitable at current scale |
| `internal/ui` | Lipgloss rendering of four screens | Mostly presentation-only |

## What the app does now

- Starts on a splash screen. The first non-`ctrl+c` key opens the test but is
  consumed rather than typed.
- Defaults to 30 seconds and derives all duration choices from `Durations` in
  `internal/test/engine.go`.
- Starts timing on the first accepted character, not when the screen opens.
- Processes typing, spaces, correction, word deletion, extra characters, and
  incorrect-word re-entry in the engine.
- Calculates WPM, raw WPM, historical keypress accuracy, consistency, and
  character counts in `internal/test/metrics.go`.
- Presents splash, test, result, and profile screens through one Bubble Tea
  model.
- Persists completed and explicitly aborted tests locally using a
  temporary-file-plus-rename save.
- Records an abort on running-test `tab`, `esc`, or `ctrl+c`. A crash or forced
  process termination is not counted because "started" is only persisted on
  finish or explicit abort.
- Keeps completed history indefinitely and derives profile aggregates by
  scanning it.

## Strong points

### Engine boundary

The engine has no UI dependencies, accepts a word generator, and exposes an
injectable clock. This is why it can be tested thoroughly with deterministic
time and word streams.

### Explicit metric semantics

The distinction between historical keypress accuracy and final word correctness
is encoded clearly. Monkeytype-specific behavior is documented where it matters
instead of emerging accidentally from UI behavior.

### Proportionate application model

The application model is still small enough that one Bubble Tea model is the
right tradeoff. Splitting every screen into a submodel now would add ceremony
without solving a current problem.

### Bounded rendering work

Only words close to the caret are laid out during rendering. This avoids
rendering the engine's complete, growing word stream.

### Simple local persistence

The temporary file is written beside the destination, allowing an atomic rename
on the normal single-process Unix path.

### Central duration extension point

Engine allocation, picker entries, and personal-best aggregation all derive
from the same duration slice. Adding another timed duration genuinely works at
the data and engine levels today.

## Findings and risks

### High: input can be accepted after the deadline

Completion is checked only when the 100 ms tick arrives in
`internal/app/app.go`. A key event arriving after the exact deadline but before
that tick is still routed to `updateTest`, and `Engine.Type` and `Engine.Space`
do not reject finished input.

This can inflate WPM, raw WPM, and accuracy. It can also make the sparkline
disagree with raw WPM: the global keypress counter accepts the late key while
the per-second bucket rejects it once its index is outside the configured
duration.

There is a related exit case: `ctrl+c` in this interval sees a finished engine,
does not record an abort, and quits before the pending tick persists the
completed result.

### High for future features: results have insufficient identity

A stored result records only `DurationSec` plus metrics. Personal bests are
therefore keyed only by duration.

Adding punctuation, numbers, another language, quote mode, or a different word
list would mix incomparable scores in personal bests and averages.

### Medium: persistence is not transactional in memory

`AddResult` and `AddAborted` mutate `Store.Data` before saving. If saving fails,
the current profile can show data that was not written durably. A later
successful save can then persist the earlier mutation unexpectedly.

The application deliberately swallows save errors so a finished test does not
crash. That is the correct availability policy, but the user receives no
warning that their statistics may not have been saved.

The full indented JSON document is also rewritten synchronously on every
result. This is acceptable for the current history size but will eventually
increase save latency as results grow.

### Medium: small-terminal behavior is incomplete

The test screen remains usable at small sizes, but the profile can be vertically
clipped. `Frame` centers content without truncation or scrolling, while the
profile contains fixed-width tables and as many as ten recent-result rows.

The duration extension point also has a visual limit: every new duration adds
another fixed-width personal-best cell.

### Medium: orchestration is barely tested

Coverage during this review was:

| Package | Statement coverage |
|---|---:|
| `internal/test` | 82.4% |
| `internal/stats` | 76.8% |
| `internal/ui` | 29.3% |
| `internal/app` | 5.6% |
| `internal/words` | 0.0% |

The missing application tests cover the riskiest integration behavior:
first-key timing, tick completion, abort persistence, duration navigation,
paste input, screen transitions, and save failure.

### Low: "pure UI" is slightly overstated

`RenderTest` accepts a live `*test.Engine` and calls time-dependent methods.
Profile rendering also calls `time.Since`. The functions do not mutate state,
but identical arguments can produce different output as wall time changes.

### Low: future language support needs display-width semantics

The engine correctly uses runes, but word wrapping uses rune count rather than
terminal cell width. Wide characters and combining marks will break alignment.
Input normalization also needs an explicit policy before non-English lists are
added.

### Minor: results help text disagrees with navigation

The results footer says `esc menu`, but `esc` creates a fresh test. The README
describes the implementation correctly.

## Remediation plan

The steps below are intentionally ordered. The early phases fix current
correctness without forcing a speculative redesign; the later phases prepare
specific future features.

### Phase 1: enforce the test deadline

1. Add a single engine-level predicate that determines whether mutating input
   is still accepted.
2. Reject `Type`, `Space`, `Backspace`, and `BackspaceWord` mutations once the
   engine has finished. Keeping this invariant in the engine prevents callers
   other than the current TUI from bypassing it.
3. In `app.Model.Update`, check for an expired running test before routing a key
   message. Finish and persist the test immediately instead of waiting for the
   next tick.
4. Define `ctrl+c` at or after the deadline as a completed test followed by
   quit, rather than an unrecorded test.
5. Add deterministic engine tests for input immediately before, exactly at, and
   immediately after the deadline.
6. Add an app test where the deadline passes between the last tick and the next
   key event.
7. Verify that WPM, raw WPM, accuracy, and `RawPerSecond` all describe the same
   accepted input set.

Acceptance criteria:

- No engine state changes at or after the configured deadline.
- A completed test cannot be lost by pressing `ctrl+c` before the next tick.
- Metric totals and the per-second series agree at the time boundary.

### Phase 2: make persistence failures coherent and visible

1. Change store writes to build a candidate `Data` value without mutating the
   live store.
2. Save that candidate to the temporary file and rename it.
3. Assign the candidate to the in-memory store only after the rename succeeds.
4. Preserve the policy that a save failure never crashes or discards the
   results screen.
5. Store a nonfatal error notice in the app model and render a short
   `stats not saved` message.
6. Clear the notice after a successful later save or when a new test begins,
   according to one documented policy.
7. Test directory-creation failure, temporary-file write failure, and rename
   failure where the platform permits deterministic setup.
8. Test that failed writes do not change the in-memory profile or totals.

Acceptance criteria:

- In-memory and on-disk state either both advance or both remain unchanged.
- Users can see that a result was not saved without losing access to it on the
  result screen.
- Existing valid JSON remains readable.

### Phase 3: add application seams and integration tests

1. Introduce the smallest store interface needed by the model, for example:

   ```go
   type ResultStore interface {
       Snapshot() stats.Data
       AddResult(stats.Result) error
       AddAborted(float64) error
   }
   ```

2. Keep `stats.Store` as the production implementation.
3. Inject the app clock used for timestamps and cursor timing instead of calling
   `time.Now` directly in multiple methods.
4. Inject or wrap engine construction so app tests receive engines with the
   same fake clock.
5. Add table-driven tests for:

   - splash-to-test transition;
   - first typing key and timer arming;
   - idle and running `esc`;
   - `tab` restart and abort accounting;
   - duration selection bounds;
   - multi-rune paste input;
   - finish-to-results persistence;
   - results/profile navigation;
   - save failure notices;
   - repeated `esc` exit hint behavior.

6. Keep one top-level Bubble Tea model until screen-specific state or key
   routing becomes materially larger.

Acceptance criteria:

- App timing and state transitions can be tested without sleeping.
- Persistence behavior can be tested without writing real files.
- The current screen flow is protected by integration tests.

### Phase 4: make rendering responsive and deterministic

1. Define supported minimum terminal dimensions and what happens below them.
2. Give each screen explicit compact, normal, and expanded layouts where
   needed.
3. On narrow profile screens:

   - stack aggregate values vertically;
   - wrap or reduce personal-best columns;
   - remove lower-priority recent-result columns;
   - reduce the number of recent rows.

4. On short profile screens, calculate the available row budget before adding
   bests and recent results.
5. Ensure the help line remains visible or deliberately replace the screen with
   a minimum-size message.
6. Pass a render timestamp or a prepared view snapshot to time-sensitive
   renderers so tests are deterministic.
7. Add a size matrix covering representative narrow, short, normal, and wide
   terminals. Assert maximum width and height with Lipgloss rather than relying
   only on golden text.
8. Retain tmux checks for final manual verification.

Acceptance criteria:

- No supported screen exceeds the terminal's width or height.
- Profile navigation remains visible at the documented minimum size.
- Render tests do not depend on wall-clock time.

### Phase 5: version stored results before adding new test types

1. Define a stable configuration value, for example:

   ```go
   type TestConfig struct {
       Mode        string
       DurationSec int
       Language    string
       WordList    string
       Punctuation bool
       Numbers     bool
   }
   ```

2. Decide which fields make two scores comparable and derive a stable
   `ModeKey` from those fields.
3. Store the configuration or mode key on every new result.
4. Add a schema version to the top-level data document.
5. Treat existing unversioned results as the current defaults:
   timed mode, English-200, no punctuation, and no numbers.
6. Implement explicit load-time migration and test fixtures for every supported
   prior schema.
7. Group personal bests by comparable mode key, not duration alone.
8. Decide whether all-time averages are global, filterable, or grouped by mode,
   and label them accordingly.
9. Move the product's available-mode catalog out of the engine if it grows
   beyond a simple duration list. The engine should accept a configuration;
   the app should decide which configurations users can choose.

Acceptance criteria:

- Old result files load without manual intervention or lost metrics.
- Scores from different languages or modifiers cannot share a personal-best
  bucket accidentally.
- Adding a new configuration dimension has a documented storage migration
  rule.

### Phase 6: prepare Unicode-aware word sources

1. Define whether targets and input are compared by rune, grapheme cluster, or
   normalized Unicode form.
2. Use terminal display width, not rune count, for line wrapping and caret
   placement.
3. Make word-list identity explicit and give every embedded list a stable ID.
4. Inject a random source when reproducibility is useful; retain random
   production behavior.
5. Test wide characters, combining marks, long single words, and extra
   characters at the wrap boundary.
6. Check repetition across generator extension boundaries if the desired rule
   changes from "within one batch" to "across the whole test."

Acceptance criteria:

- Carets and wrapping remain aligned for every supported language.
- Stored results identify the exact word source used.
- Word generation can be reproduced in tests.

### Phase 7: scale only in response to measurements

1. Add a benchmark that loads, aggregates, and saves representative large
   histories.
2. Measure synchronous save latency from the Bubble Tea update path.
3. If whole-document rewrites become visible, choose one migration:

   - append-only records plus a compact summary;
   - SQLite with a small repository implementation;
   - periodic compaction of a journal.

4. If an unlimited or very long test mode is added, stop retaining every
   historical `Word` in the active engine. Preserve aggregate metric state and
   only retain the word window needed for editing and rendering.
5. Add file locking or another single-writer strategy if multiple concurrent
   instances become supported.
6. Consider `0600` result-file permissions if local performance history is
   treated as private data.

Acceptance criteria:

- Storage migration is driven by measured latency or file-size evidence.
- The default local-only workflow remains dependency-light.
- Multiple-process behavior is either supported safely or explicitly rejected.

## Suggested implementation order

```text
deadline correctness
        ↓
persistence transaction + warning
        ↓
app clock/store seams and tests
        ↓
responsive deterministic UI
        ↓
versioned TestConfig and migrations
        ↓
new modes / languages
        ↓
storage or engine scaling, if measurements require it
```

The first four phases improve the existing application. Phase 5 is the gate for
feature expansion. Phases 6 and 7 should be undertaken only when their
corresponding product features are selected.

## Verification performed during the review

- `go test -cover ./...`: passed.
- `go vet ./...`: passed.
- `gofmt -l .`: clean.
- Production build: succeeded. The Go tool emitted only a read-only
  module-cache metadata warning in the review environment.
- Isolated tmux checks verified splash, test, profile, abort persistence, and
  narrow-terminal behavior.
