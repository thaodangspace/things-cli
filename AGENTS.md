# things-cli agent notes

`things-cli` is a macOS-only Go/Cobra CLI for Things 3. It follows the sibling CLI house style: JSON envelopes by default (`{ok,data}` / `{ok,error}`), `--human` text summaries, and exit codes 0 success / 1 runtime / 2 usage/config.

## Layout

- `cmd/things-cli/main.go` — entrypoint (`os.Exit(cli.Execute())`).
- `cli/` — Cobra command tree, output envelopes, validation, and service injection.
- `things/model.go` — storage-neutral response and request models plus the `Service` contract.
- `things/client.go` — Things automation service implementation.
- `things/automation.go` — injectable `ScriptRunner`, `/usr/bin/osascript` process runner, JSON envelope decoding, timeout and permission errors.
- `things/automation.js` — fixed JXA entrypoint using Things' public scripting dictionary.
- `things/testdata/automation/` — committed JSON response fixtures.
- `skills/things-cli/SKILL.md` — agent skill for installing/using the CLI.
- `docs/` — Astro/Starlight static documentation site, built independently from the Go CLI.

## Automation boundary

All Things reads, writes, and navigation use `osascript -l JavaScript` and bundle identifier `com.culturedcode.ThingsMac`. User-controlled values are passed as JSON process arguments and are never interpolated into JXA source. The CLI never reads or writes Things' SQLite database and has no database path or auth-token configuration.

Reads use native Things list membership/order for Inbox, Today, Upcoming, Anytime, Someday, Logbook, and Trash. General queries enumerate publicly reachable items, deduplicate by Things ID, and apply status/type/tag/area/project filters. Public automation does not expose checklist items or headings; responses retain `checklist: []` and `heading: null`.

Writes are synchronous and return affected IDs. Do not retry a write after timeout or process failure because the outcome may be indeterminate. `--wait` is a compatibility flag and does not poll storage. `search` is UI navigation; it opens an encoded Things search URL through JXA and does not return search results.

macOS Automation/TCC permission belongs to the invoking terminal, agent host, or executable. Do not attempt to bypass it. Missing app, denied permission, timeout, malformed response, and application errors must be reported as actionable runtime errors.

## Testing

Default verification is fixture-based and never mutates live Things:

```bash
make test
make test-race
make vet
make build
```

Use a local cache when needed:

```bash
mkdir -p .cache/go-build .cache/gopath
GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go-build GOPATH=$PWD/.cache/gopath go test ./...
```

The docs site uses npm from `docs/` and emits static output to `docs/dist/`:

```bash
make docs-build
```

For Cloudflare Pages, use `docs` as the root directory, `npm run build` as the
build command, and `dist` as the output directory.

The opt-in read-only smoke test requires Things 3 and Automation permission:

```bash
THINGS_LIVE_TEST=1 go test ./things -run TestLiveReadOnlyAutomation -count=1
```

Do not add live mutation tests to the default suite.
