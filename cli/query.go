package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newQueryCommand() *cobra.Command {
	var f things.Filter
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query Things todos/projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLimit(f.Limit); err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			items, err := currentThingsService().Query(ctx, f)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), items, opts.human, humanItemList(items, "No items found."))
		},
	}
	cmd.Flags().StringVar(&f.Status, "status", "", "filter by status: open|completed|canceled")
	cmd.Flags().StringVar(&f.Type, "type", "", "filter by type: to-do|project")
	cmd.Flags().StringVar(&f.Tag, "tag", "", "filter by tag id/title")
	cmd.Flags().StringVar(&f.Area, "area", "", "filter by area id/title")
	cmd.Flags().StringVar(&f.Project, "project", "", "filter by project id/title")
	cmd.Flags().IntVar(&f.Limit, "limit", 50, "maximum results (1..100)")
	return cmd
}
