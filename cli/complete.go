package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newCompleteCommand() *cobra.Command { return statusCommand("complete", "completed") }
func newCancelCommand() *cobra.Command   { return statusCommand("cancel", "canceled") }

func statusCommand(name, field string) *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:   name + " <id>",
		Short: name + " a Things todo",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("%s requires exactly one id", name)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			var res things.ActionResult
			var err error
			if field == "completed" {
				res, err = currentThingsService().Complete(ctx, args[0])
			} else {
				res, err = currentThingsService().Cancel(ctx, args[0])
			}
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "compatibility flag; automation writes are synchronous")
	return cmd
}
