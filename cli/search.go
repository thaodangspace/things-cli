package cli

import "github.com/spf13/cobra"

func newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search in Things GUI",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("search requires exactly one query")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Search(ctx, args[0])
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
}
