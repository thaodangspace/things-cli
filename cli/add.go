package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newAddCommand() *cobra.Command {
	var title, notes, when, deadline, tags, list, listID, project, projectID string
	var completed, canceled, reveal, wait bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a Things todo via macOS automation",
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := validateAddRequest(things.AddRequest{
				Title: title, Notes: notes, When: when, Deadline: deadline,
				Tags: splitCSV(tags), List: list, ListID: listID,
				Project: project, ProjectID: projectID,
				Completed: completed, Canceled: canceled, Reveal: reveal,
			})
			if err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().Add(ctx, request)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "todo title (required)")
	cmd.Flags().StringVar(&notes, "notes", "", "todo notes")
	cmd.Flags().StringVar(&when, "when", "", "schedule: today|tomorrow|evening|anytime|someday|yyyy-mm-dd[@HH:MM]")
	cmd.Flags().StringVar(&deadline, "deadline", "", "deadline date")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	cmd.Flags().StringVar(&list, "list", "", "destination list name")
	cmd.Flags().StringVar(&listID, "list-id", "", "destination list id")
	cmd.Flags().StringVar(&project, "project", "", "destination project name")
	cmd.Flags().StringVar(&projectID, "project-id", "", "destination project id")
	cmd.Flags().BoolVar(&completed, "completed", false, "create completed")
	cmd.Flags().BoolVar(&canceled, "canceled", false, "create canceled")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "reveal in Things")
	cmd.Flags().BoolVar(&wait, "wait", false, "compatibility flag; automation writes are synchronous")
	return cmd
}
