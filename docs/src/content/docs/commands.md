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
  [--tag TAG] [--area AREA] [--project PROJECT] [--limit N]
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
things-cli move <id> (--list NAME|--list-id ID|--project NAME|--project-id ID|--area NAME|--area-id ID)
things-cli detach <id> (--project|--area|--all)
things-cli delete <id> [--reveal]
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
successful deletion. `empty-trash` irreversibly deletes all items in Trash and
requires the explicit `--yes` confirmation.

`--wait` is accepted on add, add-project, update, move, and detach for
compatibility. It does not poll storage: a successful automation response
already means the operation completed. Do not retry a write after a timeout
because the outcome may be indeterminate.

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
