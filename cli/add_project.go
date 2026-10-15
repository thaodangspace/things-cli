package cli

import (
	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func newAddProjectCommand() *cobra.Command {
	var title, notes, when, deadline, tags, area, areaID, todos string
	var reveal, wait bool
	cmd := &cobra.Command{
		Use:   "add-project",
		Short: "Add a Things project via macOS automation",
		RunE: func(cmd *cobra.Command, args []string) error {
			request, err := validateAddProjectRequest(things.AddProjectRequest{
				Title: title, Notes: notes, When: when, Deadline: deadline,
				Tags: splitCSV(tags), Area: area, AreaID: areaID,
				ToDos: splitLines(todos), Reveal: reveal,
			})
			if err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			res, err := currentThingsService().AddProject(ctx, request)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), res, opts.human, summarizeAction(res))
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "project title (required)")
	cmd.Flags().StringVar(&notes, "notes", "", "project notes")
	cmd.Flags().StringVar(&when, "when", "", "schedule")
	cmd.Flags().StringVar(&deadline, "deadline", "", "deadline")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	cmd.Flags().StringVar(&area, "area", "", "area name")
	cmd.Flags().StringVar(&areaID, "area-id", "", "area id")
	cmd.Flags().StringVar(&todos, "to-dos", "", "newline-separated todos")
	cmd.Flags().BoolVar(&reveal, "reveal", false, "reveal in Things")
	cmd.Flags().BoolVar(&wait, "wait", false, "compatibility flag; automation writes are synchronous")
	return cmd
}
