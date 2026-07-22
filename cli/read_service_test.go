package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/thaodangspace/things-cli/things"
)

type readProbeService struct {
	fixtureService
	calls []string
}

func (s *readProbeService) List(context.Context, things.ListKind, int) ([]things.Item, error) {
	s.calls = append(s.calls, "list")
	return []things.Item{{ID: "automation-item", Title: "Automation item", Type: "to-do", Status: "open"}}, nil
}
func (s *readProbeService) Query(context.Context, things.Filter) ([]things.Item, error) {
	s.calls = append(s.calls, "query")
	return []things.Item{}, nil
}
func (s *readProbeService) Get(context.Context, string) (things.Item, error) {
	s.calls = append(s.calls, "get")
	return things.Item{ID: "automation-item", Title: "Automation item", Type: "to-do", Status: "open"}, nil
}
func (s *readProbeService) ListProjects(context.Context, string, int) ([]things.Item, error) {
	s.calls = append(s.calls, "list-projects")
	return []things.Item{}, nil
}
func (s *readProbeService) ListAreas(context.Context) ([]things.Area, error) {
	s.calls = append(s.calls, "list-areas")
	return []things.Area{}, nil
}
func (s *readProbeService) ListTags(context.Context) ([]things.Tag, error) {
	s.calls = append(s.calls, "list-tags")
	return []things.Tag{}, nil
}

func TestReadCommandsUseServiceBoundary(t *testing.T) {
	probe := &readProbeService{}
	old := thingsService
	thingsService = probe
	t.Cleanup(func() { thingsService = old })
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "list", args: []string{"today"}},
		{name: "query", args: []string{"query"}},
		{name: "get", args: []string{"get", "id"}},
		{name: "projects", args: []string{"list-projects"}},
		{name: "areas", args: []string{"list-areas"}},
		{name: "tags", args: []string{"list-tags"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := newRootCommand()
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetArgs(tc.args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), `"ok": true`) {
				t.Fatalf("output=%s", out.String())
			}
		})
	}
	want := []string{"list", "query", "get", "list-projects", "list-areas", "list-tags"}
	if strings.Join(probe.calls, ",") != strings.Join(want, ",") {
		t.Fatalf("calls=%v want=%v", probe.calls, want)
	}
}
