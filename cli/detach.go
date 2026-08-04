package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newDetachCommand() *cobra.Command {
	var project, area, all, wait bool
	cmd := &cobra.Command{
		Use:   "detach <id> (--project|--area|--all)",
		Short: "Detach a Things todo/project from its project or area via macOS automation",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("detach requires exactly one id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDetachScope(project, area, all); err != nil {
				return err
			}
			request := things.DetachRequest{ID: args[0], All: all}
			if project || all {
				request.Project = true
			}
			if area || all {
				request.Area = true
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Detach(ctx, request)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().BoolVar(&project, "project", false, "detach a todo from its project")
	cmd.Flags().BoolVar(&area, "area", false, "detach a todo/project from its area")
	cmd.Flags().BoolVar(&all, "all", false, "detach both project and area relationships where applicable")
	cmd.Flags().BoolVar(&wait, "wait", false, "compatibility flag; automation writes are synchronous")
	return cmd
}

func validateDetachScope(project, area, all bool) error {
	if all && (project || area) {
		return usageErrorf("--all cannot be combined with --project or --area")
	}
	if project && area {
		return usageErrorf("--project and --area cannot both be set; use --all to clear both")
	}
	if !project && !area && !all {
		return usageErrorf("detach requires exactly one of --project, --area, or --all")
	}
	return nil
}
