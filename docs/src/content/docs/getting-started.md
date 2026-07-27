---
title: Getting started
description: Install things-cli and make your first read-only request.
---

## Requirements

- macOS
- Things 3 for Mac
- Automation permission for the process invoking `things-cli`
- Go, if installing from source with `go install`

## Install the latest release

```sh
go install github.com/thaodangspace/things-cli/cmd/things-cli@latest
```

Make sure the Go install directory is on your `PATH`, then verify the binary:

```sh
things-cli --help
things-cli --version
```

## Read a Things list

Start with a read-only native list. The command returns a JSON envelope by
default:

```sh
things-cli inbox --limit 10
```

For a human-readable summary instead:

```sh
things-cli today --human
```

The first request may trigger a macOS Automation prompt. Allow the invoking
terminal or host application to control Things. If permission was denied, enable
it in **System Settings → Privacy & Security → Automation**.

## Next steps

- Use [Commands](/commands/) for the complete command groups and flags.
- Read [Output and errors](/output-and-errors/) before integrating with a script.
- Review [Things automation](/things-automation/) for the storage and permission boundary.
