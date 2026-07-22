package things

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type readFixtureRunner struct {
	fixtures map[string]json.RawMessage
	ops      []string
	requests []any
}

func newReadFixtureRunner(t *testing.T) *readFixtureRunner {
	t.Helper()
	path := filepath.Join("testdata", "automation", "read.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixtures map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	return &readFixtureRunner{fixtures: fixtures}
}

func (r *readFixtureRunner) Run(_ context.Context, operation string, request any) ([]byte, error) {
	r.ops = append(r.ops, operation)
	r.requests = append(r.requests, request)
	data, ok := r.fixtures[operation]
	if !ok {
		return []byte(`{"ok":false,"error":{"code":"missing_fixture","message":"missing fixture"}}`), nil
	}
	return json.Marshal(struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}{OK: true, Data: data})
}

func TestAutomationClientListAndGet(t *testing.T) {
	runner := newReadFixtureRunner(t)
	client := NewAutomationClient(runner)
	items, err := client.List(context.Background(), ListToday, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "todo-today" {
		t.Fatalf("items=%+v", items)
	}
	item, err := client.Get(context.Background(), "todo-today")
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "Today task" || item.Tags == nil || item.Tags[0] != "work" {
		t.Fatalf("item=%+v", item)
	}
	if !reflect.DeepEqual(runner.ops, []string{"list", "get"}) {
		t.Fatalf("operations=%v", runner.ops)
	}
}

func TestAutomationClientGetNotFound(t *testing.T) {
	runner := &fixtureRunner{response: []byte(`{"ok":false,"error":{"code":"not_found","message":"item not found: missing"}}`)}
	_, err := NewAutomationClient(runner).Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error=%v, want ErrNotFound", err)
	}
}

func TestAutomationClientQueryAndFilters(t *testing.T) {
	runner := newReadFixtureRunner(t)
	client := NewAutomationClient(runner)
	items, err := client.Query(context.Background(), Filter{Tag: "work", Limit: 7})
	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("items=%v, want fixture empty array", items)
	}
	request, ok := runner.requests[0].(queryRequest)
	if !ok || request.Tag != "work" || request.Limit != 7 {
		t.Fatalf("request=%#v", runner.requests[0])
	}
}

func TestAutomationClientMetadataAndEmptyArrays(t *testing.T) {
	runner := newReadFixtureRunner(t)
	client := NewAutomationClient(runner)
	projects, err := client.ListProjects(context.Background(), "", 3)
	if err != nil {
		t.Fatal(err)
	}
	areas, err := client.ListAreas(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tags, err := client.ListTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if projects == nil || len(projects) != 0 || len(areas) != 1 || len(tags) != 1 {
		t.Fatalf("projects=%v areas=%v tags=%v", projects, areas, tags)
	}
	if !reflect.DeepEqual(runner.ops, []string{"list-projects", "list-areas", "list-tags"}) {
		t.Fatalf("operations=%v", runner.ops)
	}
}

func TestAutomationClientValidatesFiltersBeforeDispatch(t *testing.T) {
	runner := newReadFixtureRunner(t)
	client := NewAutomationClient(runner)
	if _, err := client.Query(context.Background(), Filter{Status: "unknown"}); err == nil {
		t.Fatal("expected unknown status error")
	}
	if len(runner.ops) != 0 {
		t.Fatalf("operations=%v", runner.ops)
	}
}
