package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

func resourceService() (things.AreaTagService, error) {
	service, ok := currentThingsService().(things.AreaTagService)
	if !ok {
		return nil, usageErrorf("configured Things service does not support area and tag management")
	}
	return service, nil
}

func resourceTarget(value string) (things.ResourceTarget, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return things.ResourceTarget{}, usageErrorf("resource id or name is required")
	}
	// The automation resolver tries the ID first and then the exact name. Both
	// fields deliberately carry the same user input so names remain supported.
	return things.ResourceTarget{ID: value, Name: value}, nil
}

func resourceArg(_ *cobra.Command, args []string, resource string) (things.ResourceTarget, error) {
	if len(args) != 1 {
		return things.ResourceTarget{}, usageErrorf("%s requires exactly one id or name", resource)
	}
	return resourceTarget(args[0])
}

func runResourceAction(cmd *cobra.Command, action func(context.Context, things.AreaTagService) (things.ActionResult, error)) error {
	service, err := resourceService()
	if err != nil {
		return err
	}
	ctx, cancel := withTimeout(cmd)
	defer cancel()
	result, err := action(ctx, service)
	if err != nil {
		return err
	}
	return writeSuccess(cmd.OutOrStdout(), result, opts.human, summarizeAction(result))
}

func newAreaCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "area", Short: "Manage Things areas"}
	cmd.AddCommand(newAreaListCommand(), newAreaAddCommand(), newAreaRenameCommand(), newAreaDeleteCommand())
	return cmd
}

func newAreaListCommand() *cobra.Command {
	return &cobra.Command{
		Use: "list", Short: "List Things areas",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			areas, err := currentThingsService().ListAreas(ctx)
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), areas, opts.human, humanAreaList(areas, "No areas found."))
		},
	}
}

func newAreaAddCommand() *cobra.Command {
	var title, tags string
	cmd := &cobra.Command{
		Use: "add", Short: "Create a Things area",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name, err := requireFlag("title", title)
			if err != nil {
				return err
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.AddArea(ctx, things.AddAreaRequest{Title: name, Tags: splitCSV(tags)})
			})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "area name (required)")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated existing tag names")
	return cmd
}

func newAreaRenameCommand() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use: "rename <id-or-name>", Short: "Rename a Things area",
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := resourceArg(cmd, args, "area rename")
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resourceArg(cmd, args, "area rename")
			if err != nil {
				return err
			}
			name, err := requireFlag("title", title)
			if err != nil {
				return err
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.RenameArea(ctx, things.RenameAreaRequest{Target: target, Title: name})
			})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new area name (required)")
	return cmd
}

func newAreaDeleteCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use: "delete <id-or-name>", Short: "Delete a Things area",
		Long: "Delete an area and move its children to Trash. Requires --yes.",
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := resourceArg(cmd, args, "area delete")
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return usageErrorf("area delete requires --yes")
			}
			target, err := resourceArg(cmd, args, "area delete")
			if err != nil {
				return err
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.DeleteArea(ctx, things.DeleteAreaRequest{Target: target})
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm moving area children to Trash")
	return cmd
}

func newTagCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "tag", Short: "Manage Things tags"}
	cmd.AddCommand(newTagListCommand(), newTagAddCommand(), newTagRenameCommand(), newTagSetParentCommand(), newTagDeleteCommand())
	return cmd
}

func newTagListCommand() *cobra.Command {
	var tree bool
	cmd := &cobra.Command{
		Use: "list", Short: "List Things tags",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			var tags []things.Tag
			var err error
			if tree {
				if service, ok := currentThingsService().(things.AreaTagService); ok {
					tags, err = service.ListTagsTree(ctx)
				} else {
					// Older injected services may already return Parent on Tag;
					// retain that service boundary for compatibility.
					tags, err = currentThingsService().ListTags(ctx)
				}
			} else {
				tags, err = currentThingsService().ListTags(ctx)
			}
			if err != nil {
				return err
			}
			return writeSuccess(cmd.OutOrStdout(), tags, opts.human, humanTagListTree(tags, "No tags found."))
		},
	}
	cmd.Flags().BoolVar(&tree, "tree", false, "include parent tag relationships")
	return cmd
}

func newTagAddCommand() *cobra.Command {
	var title, parent string
	cmd := &cobra.Command{
		Use: "add", Short: "Create a Things tag", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			name, err := requireFlag("title", title)
			if err != nil {
				return err
			}
			var target things.ResourceTarget
			if strings.TrimSpace(parent) != "" {
				target, err = resourceTarget(parent)
				if err != nil {
					return err
				}
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.AddTag(ctx, things.AddTagRequest{Title: name, Parent: target})
			})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "tag name (required)")
	cmd.Flags().StringVar(&parent, "parent", "", "parent tag id or exact name")
	return cmd
}

func newTagRenameCommand() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use: "rename <id-or-name>", Short: "Rename a Things tag",
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := resourceArg(cmd, args, "tag rename")
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resourceArg(cmd, args, "tag rename")
			if err != nil {
				return err
			}
			name, err := requireFlag("title", title)
			if err != nil {
				return err
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.RenameTag(ctx, things.RenameTagRequest{Target: target, Title: name})
			})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new tag name (required)")
	return cmd
}

func newTagSetParentCommand() *cobra.Command {
	var parent string
	var root bool
	cmd := &cobra.Command{
		Use: "set-parent <id-or-name>", Short: "Set or clear a tag parent",
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := resourceArg(cmd, args, "tag set-parent")
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resourceArg(cmd, args, "tag set-parent")
			if err != nil {
				return err
			}
			if root == (strings.TrimSpace(parent) != "") {
				return usageErrorf("tag set-parent requires exactly one of --parent or --root")
			}
			var parentTarget things.ResourceTarget
			if !root {
				parentTarget, err = resourceTarget(parent)
				if err != nil {
					return err
				}
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.SetTagParent(ctx, things.SetTagParentRequest{Target: target, Parent: parentTarget, Root: root})
			})
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "parent tag id or exact name")
	cmd.Flags().BoolVar(&root, "root", false, "remove the parent and make the tag a root tag")
	return cmd
}

func newTagDeleteCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use: "delete <id-or-name>", Short: "Delete a Things tag",
		Long: "Delete a tag wherever it is used. Requires --yes.",
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := resourceArg(cmd, args, "tag delete")
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return usageErrorf("tag delete requires --yes")
			}
			target, err := resourceArg(cmd, args, "tag delete")
			if err != nil {
				return err
			}
			return runResourceAction(cmd, func(ctx context.Context, service things.AreaTagService) (things.ActionResult, error) {
				return service.DeleteTag(ctx, things.DeleteTagRequest{Target: target})
			})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm removing the tag from all items")
	return cmd
}
