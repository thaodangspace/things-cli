package cli

import (
	"strings"
	"time"

	"github.com/thaodangspace/things-cli/things"
)

func validateQueryFlags(filter things.Filter, textSet, limitSet bool) error {
	if textSet && strings.TrimSpace(filter.Text) == "" {
		return usageErrorf("--text may not be empty")
	}
	if filter.All && limitSet {
		return usageErrorf("--all cannot be combined with --limit")
	}
	if filter.Sort != "" {
		switch filter.Sort {
		case "native", "title", "created", "modified", "deadline", "start":
		default:
			return usageErrorf("unknown --sort %q; use native|title|created|modified|deadline|start", filter.Sort)
		}
	}
	for _, field := range []struct {
		name     string
		value    string
		dateOnly bool
	}{
		{"--created-after", filter.CreatedAfter, false},
		{"--created-before", filter.CreatedBefore, false},
		{"--modified-after", filter.ModifiedAfter, false},
		{"--modified-before", filter.ModifiedBefore, false},
		{"--deadline-after", filter.DeadlineAfter, true},
		{"--deadline-before", filter.DeadlineBefore, true},
		{"--start-after", filter.StartAfter, true},
		{"--start-before", filter.StartBefore, true},
	} {
		if field.value == "" {
			continue
		}
		var err error
		if field.dateOnly {
			_, err = time.Parse("2006-01-02", field.value)
		} else {
			_, err = time.Parse(time.RFC3339, field.value)
			if err != nil {
				_, err = time.Parse("2006-01-02", field.value)
			}
		}
		if err != nil {
			format := "RFC3339 or YYYY-MM-DD"
			if field.dateOnly {
				format = "YYYY-MM-DD"
			}
			return usageErrorf("invalid %s %q; use %s", field.name, field.value, format)
		}
	}
	return nil
}
