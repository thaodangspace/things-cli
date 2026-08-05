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
			limitSet := cmd.Flags().Changed("limit")
			textSet := cmd.Flags().Changed("text")
			if err := validateQueryFlags(f, textSet, limitSet); err != nil {
				return err
			}
			if f.All {
				f.Limit = 0
			} else if err := validateLimit(f.Limit); err != nil {
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
	cmd.Flags().StringVar(&f.Text, "text", "", "case-insensitive substring match against title and notes")
	cmd.Flags().StringVar(&f.CreatedAfter, "created-after", "", "creation timestamp lower bound (inclusive, RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.CreatedBefore, "created-before", "", "creation timestamp upper bound (inclusive, RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.ModifiedAfter, "modified-after", "", "modification timestamp lower bound (inclusive, RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.ModifiedBefore, "modified-before", "", "modification timestamp upper bound (inclusive, RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.DeadlineAfter, "deadline-after", "", "deadline lower bound (inclusive, YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.DeadlineBefore, "deadline-before", "", "deadline upper bound (inclusive, YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.StartAfter, "start-after", "", "start date lower bound (inclusive, YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.StartBefore, "start-before", "", "start date upper bound (inclusive, YYYY-MM-DD)")
	cmd.Flags().StringVar(&f.Sort, "sort", "", "sort by native|title|created|modified|deadline|start")
	cmd.Flags().BoolVar(&f.Reverse, "reverse", false, "reverse the final result sequence")
	cmd.Flags().BoolVar(&f.All, "all", false, "return all matching results; may be slower for broad queries")
	cmd.Flags().IntVar(&f.Limit, "limit", 50, "maximum results (1..100; mutually exclusive with --all)")
	return cmd
}
