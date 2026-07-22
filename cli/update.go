package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newUpdateCommand() *cobra.Command {
	var title, notes, prependNotes, appendNotes, when, deadline, tags, addTags, list, listID string
	var completed, canceled, project, wait bool
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a Things todo/project via macOS automation",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return usageErrorf("update requires exactly one id")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("title") && title == "" {
				return usageErrorf("--title may not be empty")
			}
			if err := validateWhen(when); err != nil {
				return err
			}
			if err := validateDeadline(deadline); err != nil {
				return err
			}
			if cmd.Flags().Changed("completed") && cmd.Flags().Changed("canceled") {
				return usageErrorf("--completed and --canceled cannot both be set")
			}
			o := things.UpdateRequest{ID: args[0], Project: project}
			setUpdateString(cmd, &o.Title, "title", title)
			setUpdateString(cmd, &o.Notes, "notes", notes)
			setUpdateString(cmd, &o.PrependNotes, "prepend-notes", prependNotes)
			setUpdateString(cmd, &o.AppendNotes, "append-notes", appendNotes)
			setUpdateString(cmd, &o.When, "when", when)
			setUpdateString(cmd, &o.Deadline, "deadline", deadline)
			setUpdateString(cmd, &o.List, "list", list)
			setUpdateString(cmd, &o.ListID, "list-id", listID)
			if cmd.Flags().Changed("tags") {
				v := splitCSV(tags)
				o.Tags = &v
			}
			if cmd.Flags().Changed("add-tags") {
				v := splitCSV(addTags)
				o.AddTags = &v
			}
			if cmd.Flags().Changed("completed") {
				o.Completed = &completed
			}
			if cmd.Flags().Changed("canceled") {
				o.Canceled = &canceled
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Update(ctx, o)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "set title (cannot be empty)")
	cmd.Flags().StringVar(&notes, "notes", "", "set notes; explicit empty clears")
	cmd.Flags().StringVar(&prependNotes, "prepend-notes", "", "prepend notes")
	cmd.Flags().StringVar(&appendNotes, "append-notes", "", "append notes")
	cmd.Flags().StringVar(&when, "when", "", "schedule")
	cmd.Flags().StringVar(&deadline, "deadline", "", "deadline")
	cmd.Flags().StringVar(&tags, "tags", "", "set comma-separated tags; explicit empty clears")
	cmd.Flags().StringVar(&addTags, "add-tags", "", "add comma-separated tags")
	cmd.Flags().StringVar(&list, "list", "", "destination list")
	cmd.Flags().StringVar(&listID, "list-id", "", "destination list id")
	cmd.Flags().BoolVar(&completed, "completed", false, "mark completed/uncompleted")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "mark canceled/uncanceled")
	cmd.Flags().BoolVar(&project, "project", false, "use update-project action")
	cmd.Flags().BoolVar(&wait, "wait", false, "compatibility flag; automation writes are synchronous")
	return cmd
}

func setUpdateString(cmd *cobra.Command, dest **string, name, value string) {
	if cmd.Flags().Changed(name) {
		v := value
		*dest = &v
	}
}
