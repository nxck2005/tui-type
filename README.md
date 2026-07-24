# tui-type

Minimalist, local-only typing test for the terminal, styled after
[monkeytype](https://monkeytype.com) (Serika Dark palette). No network, no
accounts — your stats live in a JSON file on your machine.

## Install / run

```sh
go build -o tui-type .
./tui-type
```

## Usage

Timed modes: **10 / 15 / 30 / 60 / 120** seconds (defined in one slice in
`internal/test/engine.go` — add an entry to add a mode).

| Key | Where | Action |
|---|---|---|
| any letter | test | start typing (clock starts on first keystroke) |
| `←` / `→` | test (idle) | change duration |
| `tab` | test | restart / new words |
| `esc` | test (idle) | open profile |
| `esc` | test (running) | stop the test |
| `ctrl+w` / `ctrl+h` | test | delete current word |
| `tab` / `esc` | results / profile | back to a fresh test |
| `ctrl+c` | anywhere | quit (an expired running test is saved first) |

The clock is enforced at the engine boundary: input is accepted strictly
before the deadline and ignored at or after it, even if the next 100 ms UI
tick has not arrived yet.

## Stats

The results screen shows WPM, accuracy, raw WPM, consistency, a character
breakdown (correct/incorrect/extra/missed), and a WPM-over-time sparkline.
The profile tab tracks tests started/completed, time typing, all-time
highs and averages, personal bests per duration, and your recent tests.

Metrics use monkeytype's formulas: WPM counts only correctly typed words
(chars/5 per minute), accuracy counts every keypress (corrected mistakes
still count against you), and consistency is the kogasa function over the
per-second raw WPM variance.

Data is stored at `$XDG_DATA_HOME/tui-type/results.json`
(default `~/.local/share/tui-type/results.json`). Delete the file to reset.
