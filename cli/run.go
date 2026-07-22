package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

var thingsService things.Service

func withTimeout(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return context.WithTimeout(cmd.Context(), opts.timeout)
}

func currentThingsService() things.Service {
	if thingsService == nil {
		thingsService = things.NewAutomationClient(things.ExecScript{Logger: func(operation string, duration time.Duration, err error) {
			if !opts.verbose {
				return
			}
			status := "ok"
			if err != nil {
				status = "error"
			}
			fmt.Fprintf(os.Stderr, "things automation operation=%s duration=%s status=%s\n", operation, duration.Round(time.Millisecond), status)
		}})
	}
	return thingsService
}

func validateLimit(limit int) error {
	if limit < 1 || limit > 100 {
		return usageErrorf("--limit must be between 1 and 100, got %d", limit)
	}
	return nil
}

func requireFlag(name, value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", usageErrorf("--%s is required", name)
	}
	return v, nil
}

func validateDeadline(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(v)); err != nil {
		return usageErrorf("invalid deadline %q; use yyyy-mm-dd", v)
	}
	return nil
}

func validateWhen(v string) error {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	switch s {
	case "today", "tomorrow", "anytime", "someday":
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return nil
	}
	if _, err := time.Parse("2006-01-02@15:04", s); err == nil {
		return nil
	}
	return usageErrorf("invalid date %q; use today|tomorrow|anytime|someday|yyyy-mm-dd[@HH:MM]", v)
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func splitLines(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(strings.ReplaceAll(v, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func validateCaps(title, notes string, checklist []string) error {
	if len(title) > 4000 {
		return usageErrorf("--title is too long (max 4000 bytes)")
	}
	if len(notes) > 10000 {
		return usageErrorf("--notes is too long (max 10000 bytes)")
	}
	if len(checklist) > 100 {
		return usageErrorf("checklist is too long (max 100 items)")
	}
	return nil
}
