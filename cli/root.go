// Package cli builds the things-cli command tree and translates command
// results into process exit codes.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

type usageError struct{ err error }

func (e *usageError) Error() string { return e.err.Error() }
func (e *usageError) Unwrap() error { return e.err }

func usageErrorf(format string, args ...any) error {
	return &usageError{err: fmt.Errorf(format, args...)}
}

type globalOptions struct {
	human   bool
	verbose bool
	timeout time.Duration
}

var opts globalOptions

// version is overridden at build time via -ldflags
// "-X github.com/thaodangspace/things-cli/cli.version=<value>".
var version = "dev"

func newRootCommand() *cobra.Command {
	opts = globalOptions{}
	root := &cobra.Command{
		Use:           "things-cli",
		Short:         "CLI for Things 3 operations (JSON output by default)",
		Long:          "things-cli controls Things 3 through macOS automation.\nOutput is JSON by default; pass --human for readable summaries.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &usageError{err: err} })

	pf := root.PersistentFlags()
	pf.BoolVar(&opts.human, "human", false, "print human-readable summaries instead of JSON")
	pf.BoolVar(&opts.verbose, "verbose", false, "log automation operations to stderr (payloads redacted)")
	pf.DurationVar(&opts.timeout, "timeout", 30*time.Second, "operation timeout")

	root.AddCommand(
		newQueryCommand(),
		newGetCommand(),
		newInboxCommand(), newTodayCommand(), newUpcomingCommand(), newAnytimeCommand(), newSomedayCommand(), newLogbookCommand(), newTrashCommand(),
		newListProjectsCommand(), newListAreasCommand(), newListTagsCommand(),
		newAreaCommand(), newTagCommand(),
		newAddCommand(), newAddProjectCommand(), newBatchCommand(),
		newUpdateCommand(), newMoveCommand(), newDetachCommand(), newDeleteCommand(), newEmptyTrashCommand(),
		newCompleteCommand(), newCancelCommand(),
		newShowCommand(), newSearchCommand(), newDoctorCommand(),
	)
	return root
}

func Execute() int {
	root := newRootCommand()
	err := root.Execute()
	if err == nil {
		return exitOK
	}
	var df doctorFailure
	if !errors.As(err, &df) {
		writeError(os.Stderr, err, opts.human)
	}
	return exitCodeFor(err)
}

func exitCodeFor(err error) int {
	if err == nil {
		return exitOK
	}
	var ue *usageError
	if errors.As(err, &ue) {
		return exitUsage
	}
	if msg := err.Error(); strings.HasPrefix(msg, "unknown command") || strings.HasPrefix(msg, "unknown flag") || strings.HasPrefix(msg, "unknown shorthand flag") {
		return exitUsage
	}
	return exitRuntime
}
