package cli

import "github.com/spf13/cobra"

func newListProjectsCommand() *cobra.Command {
	var area string
	var limit int
	cmd := &cobra.Command{
		Use:   "list-projects",
		Short: "List Things projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLimit(limit); err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			items, err := currentThingsService().ListProjects(ctx, area, limit)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), items, opts.human, humanItemList(items, "No projects found."))
		},
	}
	cmd.Flags().StringVar(&area, "area", "", "filter by area id/title")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum results (1..100)")
	return cmd
}
