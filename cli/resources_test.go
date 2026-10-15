package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/thaodangspace/things-cli/things"
)

type resourceServiceProbe struct {
	fixtureService
	actions []string
	last    any
}

func (s *resourceServiceProbe) ListTagsTree(context.Context) ([]things.Tag, error) {
	return []things.Tag{{ID: "tag-child", Title: "Child", Parent: &things.Ref{ID: "tag-root", Title: "Root"}}}, nil
}
func (s *resourceServiceProbe) AddArea(_ context.Context, request things.AddAreaRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "area-add")
	s.last = request
	return things.ActionResult{Action: "area-add", ID: "area-new"}, nil
}
func (s *resourceServiceProbe) RenameArea(_ context.Context, request things.RenameAreaRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "area-rename")
	s.last = request
	return things.ActionResult{Action: "area-rename", ID: "area-1"}, nil
}
func (s *resourceServiceProbe) DeleteArea(_ context.Context, request things.DeleteAreaRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "area-delete")
	s.last = request
	return things.ActionResult{Action: "area-delete", ID: "area-1"}, nil
}
func (s *resourceServiceProbe) AddTag(_ context.Context, request things.AddTagRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "tag-add")
	s.last = request
	return things.ActionResult{Action: "tag-add", ID: "tag-new"}, nil
}
func (s *resourceServiceProbe) RenameTag(_ context.Context, request things.RenameTagRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "tag-rename")
	s.last = request
	return things.ActionResult{Action: "tag-rename", ID: "tag-1"}, nil
}
func (s *resourceServiceProbe) SetTagParent(_ context.Context, request things.SetTagParentRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "tag-set-parent")
	s.last = request
	return things.ActionResult{Action: "tag-set-parent", ID: "tag-1"}, nil
}
func (s *resourceServiceProbe) DeleteTag(_ context.Context, request things.DeleteTagRequest) (things.ActionResult, error) {
	s.actions = append(s.actions, "tag-delete")
	s.last = request
	return things.ActionResult{Action: "tag-delete", ID: "tag-1"}, nil
}

func TestResourceCommandsResolveTargetsAndConfirmDeletes(t *testing.T) {
	service := &resourceServiceProbe{}
	old := thingsService
	thingsService = service
	t.Cleanup(func() { thingsService = old })

	for _, args := range [][]string{
		{"area", "add", "--title", "Work", "--tags", "home, urgent"},
		{"area", "rename", "Work", "--title", "Home"},
		{"area", "delete", "Work", "--yes"},
		{"tag", "add", "--title", "Child", "--parent", "Root"},
		{"tag", "rename", "Child", "--title", "Subtask"},
		{"tag", "set-parent", "Child", "--parent", "Root"},
		{"tag", "set-parent", "Child", "--root"},
		{"tag", "delete", "Child", "--yes"},
	} {
		root := newRootCommand()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("args=%v: %v", args, err)
		}
	}
	if got, want := strings.Join(service.actions, ","), "area-add,area-rename,area-delete,tag-add,tag-rename,tag-set-parent,tag-set-parent,tag-delete"; got != want {
		t.Fatalf("actions=%q want %q", got, want)
	}
}

func TestResourceCommandsRequireDestructiveConfirmation(t *testing.T) {
	service := &resourceServiceProbe{}
	old := thingsService
	thingsService = service
	t.Cleanup(func() { thingsService = old })
	for _, args := range [][]string{{"area", "delete", "Work"}, {"tag", "delete", "Work"}} {
		root := newRootCommand()
		root.SetArgs(args)
		if err := root.Execute(); exitCodeFor(err) != exitUsage {
			t.Fatalf("args=%v err=%v", args, err)
		}
	}
	if len(service.actions) != 0 {
		t.Fatalf("destructive actions were invoked: %v", service.actions)
	}
}

func TestTagListTreeOutput(t *testing.T) {
	service := &resourceServiceProbe{}
	old := thingsService
	thingsService = service
	t.Cleanup(func() { thingsService = old })
	root := newRootCommand()
	root.SetArgs([]string{"tag", "list", "--tree", "--human"})
	var out strings.Builder
	root.SetOut(&out)
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "parent: tag-root Root") {
		t.Fatalf("output=%q", out.String())
	}
}
