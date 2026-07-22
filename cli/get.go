package cli

import (
	"github.com/spf13/cobra"
)

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get one todo/project by Things uuid",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("get requires exactly one id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			item, err := currentThingsService().Get(ctx, args[0])
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), item, opts.human, summarizeItem(item))
		},
	}
}
