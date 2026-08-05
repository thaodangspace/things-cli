---
title: Commands
description: Read, query, navigate, and modify Things from the shell.
---

Global options can be used with every command:

- `--human` prints a text summary instead of JSON.
- `--verbose` logs sanitized automation diagnostics to stderr; user payloads are not logged.
- `--timeout duration` sets the operation timeout. The default is 30 seconds.

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
```

## Write commands

Writes are synchronous and return the affected Things ID when successful:

```sh
things-cli add --title "Prepare release" [options]
things-cli add-project --title "Release" [options]
things-cli update <id> [options]
things-cli complete <id>
things-cli cancel <id>
```

Common `add` options include `--notes`, `--when`, `--deadline`, `--tags`,
`--list`, `--list-id`, `--project`, and `--project-id`. `add-project` accepts
`--to-dos` as newline-separated todo titles and can target an area with `--area`.

`--wait` is accepted on add and add-project for compatibility. It does not poll
storage: a successful automation response already means the operation completed.
Do not retry a write after a timeout because the outcome may be indeterminate.

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
