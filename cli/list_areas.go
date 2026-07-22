package cli

import "github.com/spf13/cobra"

func newListAreasCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-areas",
		Short: "List Things areas",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			areas, err := currentThingsService().ListAreas(ctx)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), areas, opts.human, humanAreaList(areas, "No areas found."))
		},
	}
}
