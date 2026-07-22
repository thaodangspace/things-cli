package cli

import "github.com/spf13/cobra"

func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id-or-list>",
		Short: "Reveal an item/list in Things",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("show requires exactly one id-or-list")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Show(ctx, args[0])
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
}
