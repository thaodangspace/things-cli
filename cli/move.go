package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newMoveCommand() *cobra.Command {
	var list, listID, project, projectID, area, areaID string
	var wait bool
	cmd := &cobra.Command{
		Use:   "move <id> (--list NAME|--list-id ID|--project NAME|--project-id ID|--area NAME|--area-id ID)",
		Short: "Move a Things todo/project to a list, project, or area via macOS automation",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("move requires exactly one id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateMoveDestinationFlags(cmd, list, listID, project, projectID, area, areaID); err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Move(ctx, things.MoveRequest{
				ID:        args[0],
				List:      list,
				ListID:    listID,
				Project:   project,
				ProjectID: projectID,
				Area:      area,
				AreaID:    areaID,
			})
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().StringVar(&list, "list", "", "destination list name (not Upcoming)")
	cmd.Flags().StringVar(&listID, "list-id", "", "destination list id")
	cmd.Flags().StringVar(&project, "project", "", "destination project name")
	cmd.Flags().StringVar(&projectID, "project-id", "", "destination project id")
	cmd.Flags().StringVar(&area, "area", "", "destination area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "destination area id")
	cmd.Flags().BoolVar(&wait, "wait", false, "compatibility flag; automation writes are synchronous")
	return cmd
}

// validateMoveDestinationFlags enforces that exactly one destination flag is
// supplied. Names and IDs are alternatives, not two destinations; rejecting
// both also prevents silently ignoring a typo in one of the flags.
func validateMoveDestinationFlags(cmd *cobra.Command, list, listID, project, projectID, area, areaID string) error {
	flags := []struct {
		name  string
		value string
	}{
		{"--list", list}, {"--list-id", listID},
		{"--project", project}, {"--project-id", projectID},
		{"--area", area}, {"--area-id", areaID},
	}
	changed := 0
	for _, flag := range flags {
		if cmd.Flags().Changed(strings.TrimPrefix(flag.name, "--")) {
			changed++
		}
	}
	if changed == 0 {
		return usageErrorf("move requires exactly one of --list, --list-id, --project, --project-id, --area, or --area-id")
	}
	if changed > 1 {
		return usageErrorf("move accepts exactly one destination flag")
	}
	return validateMoveDestination(list, listID, project, projectID, area, areaID)
}

// validateMoveDestination validates a request independently of Cobra. It is
// also a second line of defense for callers that construct requests directly.
func validateMoveDestination(list, listID, project, projectID, area, areaID string) error {
	count := 0
	if list != "" || listID != "" {
		count++
	}
	if project != "" || projectID != "" {
		count++
	}
	if area != "" || areaID != "" {
		count++
	}
	if count == 0 {
		return usageErrorf("move requires a non-empty destination")
	}
	if count > 1 {
		return usageErrorf("move accepts exactly one destination")
	}
	if strings.EqualFold(strings.TrimSpace(list), "upcoming") || strings.EqualFold(strings.TrimSpace(listID), "TMCalendarListSource") {
		return usageErrorf("moving directly to Upcoming is not supported; use update --when to schedule")
	}
	return nil
}
