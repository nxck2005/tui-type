# tui-type — agent/contributor context

Minimalist, local-only, monkeytype-style typing test TUI in Go. No network,
no accounts; stats persist to a local JSON file. Built July 2026 against
Bubble Tea **v1.3** and Lipgloss **v1.1** — the v1 APIs (`tea.KeyMsg`,
`msg.Type`, `tea.KeyRunes`), *not* the v2 beta (`tea.KeyPressMsg`). Don't
"upgrade" imports without porting the key handling.

## Commands

```sh
go build -o tui-type .   # binary is gitignored
go test ./...            # engine + stats tests; all pure logic, no TUI mocks
go vet ./... && gofmt -l .
```

## Architecture (dependency direction: app → test/stats/ui → nothing)

- `internal/test` — the engine. **Pure logic, zero UI imports.** Keystroke
  processing (`engine.go`) and metrics (`metrics.go`). `Engine.Now` is an
  injectable clock; every test uses it (see `newTestEngine` in
  `engine_test.go`). Keep it pure — this is what makes the project testable.
- `internal/words` — `go:embed`ed word list (`english.txt` = monkeytype's
  english-200, all lowercase incl. "i"). `Random(n)` never repeats a word
  back-to-back within a batch.
- `internal/stats` — JSON persistence (`store.go`, atomic tmp+rename write)
  and profile aggregation (`aggregate.go`). Path: `$XDG_DATA_HOME/tui-type/
  results.json`, fallback `~/.local/share/…`.
- `internal/ui` — pure render functions (state in, styled string out). No
  model logic. Serika Dark palette in `theme.go`; `Frame()` centers content
  and pins the help line. Below `MinWidth`×`MinHeight` (`toosmall.go`,
  60×18) every screen is replaced by a btop-style prompt — gated in one
  place at the top of `Model.View`.
- `internal/app` — the single Bubble Tea model: screen enum (test/result/
  profile), key routing, tick loop (100ms, armed on first keystroke),
  persistence calls.

**The extension point the whole design hangs off:** `Durations` slice in
`internal/test/engine.go`. Add an int there → picker, PB table, and storage
all adapt. Don't hardcode durations anywhere else.

## Monkeytype semantics (deliberate, don't "fix")

- **WPM** counts only fully-correct committed words (len+1 for the space)
  plus the in-progress word while it's still a correct prefix — chars/5 per
  minute. A partially-correct committed word contributes **zero**.
- **Accuracy** counts every keypress; corrected mistakes still count
  against you. Backspace is not a keypress.
- **Consistency** = `kogasa(cov)` over per-second raw WPM — monkeytype's
  exact function (`100*(1-tanh(cov+cov³/3+cov⁵/5))`). Bursty input (e.g.
  tmux paste) correctly yields ~0%.
- Space with nothing typed is a no-op. Backspace steps back into a
  committed word only if it was *incorrect*. Extra chars per word capped at
  `maxExtra` (12).
- Aborted tests (esc/tab/ctrl+c mid-test) count toward "tests started" and
  time typing but produce no Result. Store failures on finish are swallowed
  on purpose — a lost save must not crash a finished test.

## Gotchas learned the hard way

- **Manual verification must go through tmux** (the app needs a TTY):
  ```sh
  tmux new-session -d -s tt -x 100 -y 28 "env XDG_DATA_HOME=/tmp/somewhere ./tui-type"
  tmux send-keys -t tt -l 'the words to type '   # -l = literal, keeps spaces
  tmux capture-pane -t tt -p
  ```
  Always point `XDG_DATA_HOME` at a temp dir or you'll pollute real stats.
- `tmux capture-pane` output can *look* vertically uncentered because blank
  lines get visually trimmed downstream. Before chasing a centering bug,
  count lines: `capture-pane -p | grep -n .` — the layout was fine.
- **Word-wrap width must include the caret cell**: when the caret hangs past
  a word's end (`len(Typed) >= len(Target)`), the rendered word is one cell
  wider. `wordWidth()` in `ui/testview.go` accounts for this; forgetting it
  causes off-by-one line overflow while typing extras.
- **`lipgloss.JoinVertical(lipgloss.Center, …)` re-centers each line
  independently** — table rows of different widths shift against each other.
  Fix: pad the *last* column too so all rows are equal width (see `row()`
  in `ui/profileview.go`). Same trap for the dot-joined stat rows: a single
  long line wider than the frame wraps mid-item and each fragment recenters.
  Fix: `flowJoin()` packs items into lines ≤ `width-2` before centering.
- Only ~40 words past `Engine.Cur` are laid out per frame (`renderWords`
  `limit`) — the word list grows unboundedly during a test; don't render it
  all.
- The terminal's own background is intentionally left unset (no `#323437`
  fill) — painting every cell causes artifacts and fights user themes.

## Conventions

- gofmt-clean is enforced by habit; run it before finishing.
- Engine tests use `fixedGen(...)` for deterministic words and advance the
  fake clock via the returned `*time.Time`.
- Comments state constraints/behaviors the code can't show (monkeytype
  parity rules), not narration.

## Known deliberate scope cuts (fine to add later)

Punctuation/numbers toggles, multiple themes, other languages/word lists,
word-count & quote modes. The engine treats the word stream generically, so
these slot in via `words` + a small amount of picker UI.

# Guidelines

## Usage and efficiency rules

- Read only files relevant to the requested task.
- Do not scan the entire repository unless necessary.
- Start with targeted searches such as `rg`, `git status`, and `git diff`.
- Prefer focused tests before running the complete test suite.
- Batch related edits into one implementation pass.
- Do not repeatedly summarize unchanged context.
- Keep explanations concise and avoid printing complete files.
- Do not investigate unrelated warnings or pre-existing failures.
- Stop once the stated acceptance criteria are satisfied.
- Ask for clarification only when proceeding would likely cause incorrect work.
- Escalate model or reasoning effort only after the current setting fails or the task is clearly complex.
- Never use Max, Extra High, or Ultra reasoning for routine edits.