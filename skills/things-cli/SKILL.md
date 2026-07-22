---
name: things-cli
description: Use the things-cli command-line tool to read and manipulate Things 3 tasks/projects on macOS. Use when the user asks to inspect Things lists/tasks, query by status/type/tag/area/project, list areas/tags/projects, add or update tasks/projects, complete/cancel tasks, or open Things UI from the shell.
---

# Things CLI

Use the locally installed `things-cli` executable to automate Things 3 on macOS. It invokes Things through the public scripting dictionary with `osascript -l JavaScript`; it never reads or writes Things' SQLite database and does not require an auth token.

The invoking terminal, agent host, or executable must have Automation permission for Things 3 in **System Settings → Privacy & Security → Automation**. Missing Things, denied permission, malformed responses, application errors, and timeouts are reported as actionable errors.

## Output and global options

JSON is the default output, using `{ "ok": true, "data": ... }` on success and `{ "ok": false, "error": { "message": ... } }` on failure. Add `--human` for concise text summaries. `--verbose` logs sanitized operation diagnostics to stderr; user payloads are redacted. `--timeout 30s` controls the operation timeout (30 seconds by default). `--version` prints the build version.

Exit codes:

- `0` — success
- `1` — runtime, Things, permission, or timeout error
- `2` — usage/configuration error

If the executable is not on `PATH`, install it with:

```bash
go install github.com/thaodangspace/things-cli/cmd/things-cli@latest
```

## Read operations

Native list commands preserve Things' own list membership and ordering. Each accepts `--limit N` (default 50, maximum 100):

```bash
things-cli inbox [--limit N]
things-cli today [--limit N]
things-cli upcoming [--limit N]
things-cli anytime [--limit N]
things-cli someday [--limit N]
things-cli logbook [--limit N]
things-cli trash [--limit N]
```

Get one task or project by Things UUID:

```bash
things-cli get <id>
```

Query publicly reachable tasks/projects, deduplicated by Things ID:

```bash
things-cli query [--status open|completed|canceled] \
  [--type to-do|project] [--tag ID-or-title] [--area ID-or-title] \
  [--project ID-or-title] [--limit N]
```

`--status` also accepts `done` and `cancelled`; `--type` accepts `todo` and `task` as aliases. List metadata:

```bash
things-cli list-projects [--area ID-or-title] [--limit N]
things-cli list-areas
things-cli list-tags
```

The public automation API does not expose checklist items or headings. Responses therefore retain `checklist: []` and `heading: null`.

## Write and navigation operations

Writes are synchronous and return an affected ID. `--wait` is accepted for compatibility only; it does not poll storage or wait for eventual consistency.

Create a todo:

```bash
things-cli add --title "Follow up" [--notes "..."] \
  [--when today|tomorrow|anytime|someday|YYYY-MM-DD[@HH:MM]] \
  [--deadline YYYY-MM-DD] [--tags work,urgent] \
  [--list LIST-NAME|--list-id LIST-ID] \
  [--completed|--canceled] [--reveal] [--wait]
```

Create a project, optionally assigning an area and newline-separated todos:

```bash
things-cli add-project --title "Plan launch" [--notes "..."] \
  [--when SCHEDULE] [--deadline YYYY-MM-DD] [--tags work] \
  [--area AREA-NAME|--area-id AREA-ID] \
  [--to-dos $'Draft\nReview'] [--reveal] [--wait]
```

Update a task or project. Only supplied flags are changed; explicit empty `--notes` or `--tags` clears that value:

```bash
things-cli update <id> [--title TITLE] [--notes NOTES] \
  [--prepend-notes NOTES|--append-notes NOTES] [--when SCHEDULE] \
  [--deadline YYYY-MM-DD] [--tags TAG1,TAG2] [--add-tags TAG1,TAG2] \
  [--list LIST-NAME|--list-id LIST-ID] [--completed|--canceled] \
  [--project] [--wait]
```

`--project` selects the Things project update action. Other status/navigation commands are:

```bash
things-cli complete <id> [--wait]
things-cli cancel <id> [--wait]
things-cli show <id-or-list>
things-cli search "query"
```

`show` reveals an item or native list in the Things UI. `search` opens the Things search UI and does not return search results.

## Safety and automation behavior

User-controlled values are passed as JSON process arguments and are never interpolated into JXA source. Do not retry a write after a timeout or process failure: the outcome may be indeterminate. The CLI does not bypass macOS Automation/TCC permissions.

Checklist items, headings, and the `evening` schedule value are unsupported by the public scripting API, even though legacy-compatible flags may appear in older usage documentation.
