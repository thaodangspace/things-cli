package cli

import "github.com/spf13/cobra"

func newListTagsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-tags",
		Short: "List Things tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			tags, err := currentThingsService().ListTags(ctx)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), tags, opts.human, humanTagList(tags, "No tags found."))
		},
	}
}
