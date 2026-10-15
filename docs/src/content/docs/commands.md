---
title: Commands
description: Read, query, navigate, and modify Things from the shell.
---

Global options can be used with every command:

- `--human` prints a text summary instead of JSON.
- `--verbose` logs sanitized automation diagnostics to stderr; user payloads are not logged.
- `--timeout duration` sets the operation timeout. The default is 30 seconds.

## Diagnostics

Use the read-only doctor command to check the host before running other
commands:

```sh
things-cli doctor [--json]
things-cli doctor --human
```

The report checks macOS, `/usr/bin/osascript`, Things availability,
Automation/TCC permission, the JSON automation protocol, and the CLI version.
It never changes Things data or opens System Settings. Failed checks return exit
code 1 and include actionable remediation while retaining all check results. A
healthy JSON response starts like this:

```json
{"ok":true,"data":{"healthy":true,"version":"...","checks":[{"name":"platform","status":"pass","message":"macOS arm64"}]}}
```

## Read commands

These commands use Things' native list membership and ordering:

```sh
things-cli inbox [--limit N]
things-cli today [--limit N]
things-cli upcoming [--limit N]
things-cli anytime [--limit N]
things-cli someday [--limit N]
things-cli logbook [--limit N]
things-cli trash [--limit N]
```

Use general queries when you need filters across publicly reachable items:

```sh
things-cli query \
  [--status open|completed|canceled] \
  [--type to-do|project] \
  [--tag TAG] [--area AREA] [--project PROJECT] [--text QUERY] \
  [--created-after RFC3339] [--created-before RFC3339] \
  [--modified-after RFC3339] [--modified-before RFC3339] \
  [--deadline-after YYYY-MM-DD] [--deadline-before YYYY-MM-DD] \
  [--start-after YYYY-MM-DD] [--start-before YYYY-MM-DD] \
  [--sort native|title|created|modified|deadline|start] [--reverse] \
  [--limit N | --all]
```

Query text is a case-insensitive substring match against title and notes; it returns data and does not open the Things search UI. Creation and modification bounds accept RFC3339 (or a date-only value interpreted as local midnight). Deadline and start bounds use inclusive `YYYY-MM-DD` calendar comparisons; items without the relevant date do not match. The default is 50 results. Use `--all` to intentionally remove the cap, though broad queries may be slower because the public scripting API enumerates Things items. `--all` cannot be combined with `--limit`. Sorting is stable with ID tie-breaking; `native` preserves native list order and `--reverse` reverses the final sequence.

Examples:

```sh
things-cli query --modified-after 2026-08-01T00:00:00+07:00 --all
things-cli query --deadline-before 2026-08-07 --status open --sort deadline
things-cli query --text "quarterly review" --type to-do
```

Look up metadata and individual items with:

```sh
things-cli get <id>
things-cli list-projects [--area AREA] [--limit N]
things-cli list-areas
things-cli list-tags
things-cli area list
things-cli tag list [--tree]
```

## Write commands

Writes are synchronous and return the affected Things ID when successful:

```sh
things-cli add --title "Prepare release" [options]
things-cli add-project --title "Release" [options]
things-cli update <id> [options]
things-cli move <id> (--list NAME|--list-id ID|--project NAME|--project-id ID|--area NAME|--area-id ID)
things-cli detach <id> (--project|--area|--all)
things-cli delete <id> [--reveal]
things-cli area add --title NAME [--tags TAG1,TAG2]
things-cli area rename <id-or-name> --title NEW_NAME
things-cli area delete <id-or-name> --yes
things-cli tag add --title NAME [--parent ID_OR_NAME]
things-cli tag rename <id-or-name> --title NEW_NAME
things-cli tag set-parent <id-or-name> (--parent ID_OR_NAME | --root)
things-cli tag delete <id-or-name> --yes
things-cli empty-trash --yes
things-cli complete <id>
things-cli cancel <id>
```

Common `add` options include `--notes`, `--when`, `--deadline`, `--tags`,
`--list`, `--list-id`, `--project`, and `--project-id`. `add-project` accepts
`--to-dos` as newline-separated todo titles and can target an area with `--area`.

`move` relocates an item to exactly one destination. A to-do can move to a
built-in list, a project, or an area; a project can move to a built-in list or
an area but never into another project. Moving directly into Upcoming is not
supported — schedule with `update --when ...` instead.

`detach` removes an item's relationships: `--project` detaches a to-do from its
project, `--area` detaches a to-do or project from its area, and `--all` clears
both where applicable.

`delete` moves a to-do or project to Things Trash. Deleting a project also
moves its children to Trash. Use `--reveal` to show the Trash list after a
successful deletion. `area delete` moves the area's children to Trash, while
`tag delete` removes the tag wherever it is used; both require `--yes`. Area
`--tags` values must refer to existing tags. `empty-trash` irreversibly deletes
all items in Trash and requires the explicit `--yes` confirmation.

`--wait` is accepted on add, add-project, update, move, and detach for
compatibility. It does not poll storage: a successful automation response
already means the operation completed. Do not retry a write after a timeout
because the outcome may be indeterminate.

### Batch NDJSON

Use `batch` for generated plans and shell pipelines:

```sh
things-cli batch [--input FILE|-] [--continue-on-error] [--validate-only]

printf '%s\n' \
  '{"client_id":"task-1","operation":"add","request":{"title":"Book flights","when":"today"}}' \
  '{"client_id":"task-2","operation":"complete","request":{"id":"TODO_ID"}}' \
  | things-cli batch
things-cli batch --input plan.ndjson --validate-only
```

Input is newline-delimited JSON; blank lines are ignored. Supported operations
are `add`, `add-project`, `update`, `complete`, and `cancel`. Each operation is
validated and then executed sequentially, never concurrently or automatically
retried. Results are compact NDJSON objects containing `index`, optional
`client_id`, `ok`, and either `data` or an `{code, message}` error. The default
stops after the first failure; `--continue-on-error` emits the failure and
continues, but still exits non-zero if any operation fails. Syntax/schema
failures exit 2, while Things/runtime failures exit 1.

`--validate-only` validates the complete stream without calling Things and
emits normalized requests. `--human` is intentionally unsupported for this
stream protocol. Lines are limited to 1 MiB and streams to 1,000 operations.
A timeout or process failure can leave a write's outcome indeterminate; batch
never retries it.

## Navigation commands

```sh
things-cli show <id-or-list>
things-cli search "query"
```

`show` reveals an item or native list in Things. `search` opens Things' search UI;
it does not return search results.

:::note[Unsupported database-only concepts]
The public Things scripting dictionary does not expose checklist items or
headings. Those options are intentionally unsupported; responses use
`checklist: []` and `heading: null` for compatibility. The `evening` schedule
value is also unsupported.
:::
