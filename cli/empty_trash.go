package cli

import "github.com/spf13/cobra"

func newEmptyTrashCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "empty-trash --yes",
		Short: "Irreversibly empty the Things Trash",
		Long:  "Irreversibly delete every item currently in Things Trash. This command requires the explicit --yes flag.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return usageErrorf("empty-trash does not accept positional arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				return usageErrorf("empty-trash is irreversible; pass --yes to continue")
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().EmptyTrash(ctx)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm irreversible deletion of all items in Trash")
	return cmd
}
