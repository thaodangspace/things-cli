# things-cli

A macOS-only CLI for Things 3, designed for agents and scripts. It controls Things through the public macOS automation dictionary using `osascript -l JavaScript` (JXA). Output is JSON by default and wrapped as `{ "ok": true, "data": ... }`; pass `--human` for readable summaries.

## Requirements and permissions

- macOS with Things 3 installed.
- Automation permission for the terminal, agent host, or executable invoking `things-cli`.

On first use, macOS may ask whether the invoking application may control Things. If permission was denied, enable it under **System Settings → Privacy & Security → Automation**.

The CLI does **not** read or write Things' SQLite database and does not require a Things authentication token.

## Install

```bash
go install github.com/thaodangspace/things-cli/cmd/things-cli@latest
```

From this repository:

```bash
make build
./things-cli --help
```

## Commands

Read commands use Things' native list membership and ordering:

```bash
things-cli inbox|today|upcoming|anytime|someday|logbook|trash [--limit N]
things-cli query [--status open|completed|canceled] [--type to-do|project] [--tag TAG] [--area AREA] [--project PROJECT] [--limit N]
things-cli get <id>
things-cli list-projects [--area AREA] [--limit N]
things-cli list-areas
things-cli list-tags
```

Automation writes are synchronous and return the affected Things ID:

```bash
things-cli add --title "Task" [--notes ...] [--when today] [--deadline yyyy-mm-dd] [--tags a,b] \
  [--list LIST-NAME|--list-id LIST-ID] [--project PROJECT|--project-id PROJECT-ID] [--wait]
things-cli add-project --title "Project" [--to-dos $'one\ntwo'] [--area Work] [--wait]
things-cli update <id> --title "New" --completed
things-cli move <id> --project "Launch"
things-cli detach <id> --project
things-cli delete <id> [--reveal]
things-cli empty-trash --yes
things-cli complete <id>
things-cli cancel <id>
things-cli batch [--input FILE|-] [--continue-on-error] [--validate-only]
things-cli show <id-or-list>
things-cli search "query"
```

`delete` moves a todo or project to Things Trash; deleting a project also moves its children to Trash. `empty-trash` irreversibly deletes everything in Trash and requires the explicit `--yes` confirmation. `--reveal` on `delete` opens the Trash list after deletion.

`--wait` remains accepted for compatibility but does not poll storage; a successful automation response already means the operation completed. Do not retry a write after a timeout or process failure because the outcome may be indeterminate. The following database-dependent options are intentionally unsupported: checklist items, headings, and the `evening` schedule value. The JSON read shape keeps `heading: null` and `checklist: []` because those fields are not exposed by Things' public scripting dictionary.

`search` opens Things' search UI and does not return search results. `show` reveals an item or native list.

## Batch NDJSON

`batch` reads one JSON object per non-empty input line (stdin by default) and
writes one compact JSON result per operation to stdout. It supports `add`,
`add-project`, `update`, `complete`, and `cancel`:

```sh
printf '%s\n' \
  '{"client_id":"task-1","operation":"add","request":{"title":"Book flights","when":"today"}}' \
  '{"client_id":"task-2","operation":"complete","request":{"id":"TODO_ID"}}' \
  | things-cli batch
things-cli batch --input plan.ndjson --validate-only
```

Operations run sequentially and stop at the first failure by default. Use
`--continue-on-error` for partial execution; the exit code remains non-zero if
any operation fails. `--validate-only` reads and validates the complete stream,
invokes no Things automation, and emits normalized requests. Blank lines are
ignored. Each line is limited to 1 MiB and each stream to 1,000 operations.
Batch does not support `--human`. Writes are never retried: after a timeout or
process failure, the outcome may be indeterminate.

## Output and errors

- Exit code `0`: success.
- Exit code `1`: runtime, application, permission, or timeout error.
- Exit code `2`: usage/configuration error.

Use `--verbose` for sanitized automation operation diagnostics. User payloads are not logged. A write timeout may have an indeterminate outcome and is never retried automatically.

## Documentation

The documentation site is an Astro/Starlight app under `docs/`. Run it locally
with `make docs-dev`, or build the static site with `make docs-build`. Cloudflare
Pages deployment settings are documented in [`docs/README.md`](docs/README.md).

## Development

```bash
make test
make test-race
make vet
make build
```

The default suite uses fake automation runners and JSON fixtures. It never mutates a live Things library. Run the read-only macOS smoke test explicitly when Things is installed and permissioned:

```bash
THINGS_LIVE_TEST=1 go test ./things -run TestLiveReadOnlyAutomation -count=1
```

The production boundary is `things.Service` plus the injectable `things.ScriptRunner`; JXA source is kept in `things/automation.js`.
