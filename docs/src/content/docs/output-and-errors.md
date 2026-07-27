---
title: Output and errors
description: Integrate things-cli reliably into scripts and agent workflows.
---

## JSON envelopes

JSON is the default output format. Successful commands return an envelope shaped
like:

```json
{
  "ok": true,
  "data": {}
}
```

Failures return an error envelope:

```json
{
  "ok": false,
  "error": {
    "message": "..."
  }
}
```

Use `--human` only when the output is intended for a person rather than another
program. Diagnostics from `--verbose` go to stderr, so stdout remains suitable
for parsing.

## Exit codes

| Code | Meaning |
| ---: | --- |
| `0` | Success |
| `1` | Runtime, Things application, permission, or timeout error |
| `2` | Usage or configuration error |

Check the exit code as well as the JSON `ok` field. A command can fail before a
normal response is available, for example when its arguments are invalid or
Things cannot be reached.

## Runtime failures

Errors for a missing Things app, denied Automation permission, timeout, malformed
response, and Things application failure include actionable context. Fix the
reported condition and run the command again.

Writes are not retried automatically. A timeout or process failure can have an
indeterminate outcome, so inspect Things before deciding whether a write should
be attempted again.
