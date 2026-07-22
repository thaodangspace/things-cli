package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newListCommand(name string, kind things.ListKind) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   name,
		Short: "List Things " + name + " items",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateLimit(limit); err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			items, err := currentThingsService().List(ctx, kind, limit)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), items, opts.human, humanItemList(items, "No items found."))
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum results (1..100)")
	return cmd
}

func newInboxCommand() *cobra.Command    { return newListCommand("inbox", things.ListInbox) }
func newTodayCommand() *cobra.Command    { return newListCommand("today", things.ListToday) }
func newUpcomingCommand() *cobra.Command { return newListCommand("upcoming", things.ListUpcoming) }
func newAnytimeCommand() *cobra.Command  { return newListCommand("anytime", things.ListAnytime) }
func newSomedayCommand() *cobra.Command  { return newListCommand("someday", things.ListSomeday) }
func newLogbookCommand() *cobra.Command  { return newListCommand("logbook", things.ListLogbook) }
func newTrashCommand() *cobra.Command    { return newListCommand("trash", things.ListTrash) }
