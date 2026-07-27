---
title: Things automation

description: Understand the public automation boundary and required macOS permissions.
---

`things-cli` talks to Things with `osascript -l JavaScript` and the bundle
identifier `com.culturedcode.ThingsMac`. This is the same public scripting
interface available to macOS automation clients.

## What the CLI does not access

- It never reads or writes Things' SQLite database.
- It has no database path or authentication-token configuration.
- It does not bypass macOS Automation/TCC permission.
- It cannot expose checklist items or headings through the public dictionary.

User-controlled values are passed as JSON process arguments rather than
interpolated into JXA source.

## Permission troubleshooting

Automation permission belongs to the process that invokes the CLI. If Things is
missing, permission is denied, or the response is unavailable:

1. Confirm that Things 3 is installed and running if needed.
2. Retry from the same terminal or host application that will run the automation.
3. Open **System Settings → Privacy & Security → Automation**.
4. Allow that invoking application to control Things 3.
5. Run a read-only command such as `things-cli inbox --limit 1`.

The opt-in live test is read-only and requires both Things 3 and permission:

```sh
THINGS_LIVE_TEST=1 go test ./things -run TestLiveReadOnlyAutomation -count=1
```

The normal test suite uses fixtures and never mutates a live Things library.
