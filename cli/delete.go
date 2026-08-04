package cli

import "github.com/spf13/cobra"

func newDeleteCommand() *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "delete <id> [--reveal]",
		Short: "Move a Things todo/project to Trash via macOS automation",
		Long:  "Move a Things todo/project to Trash. Deleting a project also moves its children to Trash.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("delete requires exactly one id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Delete(ctx, args[0])
			if err != nil {
				return err
			}
			if reveal {
				if _, err := currentThingsService().Show(ctx, "trash"); err != nil {
					return err
				}
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "reveal the Trash list after deletion")
	return cmd
}
