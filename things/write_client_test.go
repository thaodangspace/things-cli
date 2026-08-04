package things

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

type actionRunner struct {
	operations []string
	requests   []any
}

func (r *actionRunner) Run(_ context.Context, operation string, request any) ([]byte, error) {
	r.operations = append(r.operations, operation)
	r.requests = append(r.requests, request)
	id := "created-id"
	if operation == "update" || operation == "complete" || operation == "cancel" {
		id = "existing-id"
	}
	return json.Marshal(map[string]any{
		"ok":   true,
		"data": ActionResult{Action: operation, ID: id},
	})
}

func TestAutomationClientWriteRequests(t *testing.T) {
	runner := &actionRunner{}
	client := NewAutomationClient(runner)
	ctx := context.Background()
	if got, err := client.Add(ctx, AddRequest{Title: "Mua sữa", Tags: []string{"home"}}); err != nil || got.ID != "created-id" {
		t.Fatalf("add=%+v err=%v", got, err)
	}
	if got, err := client.Add(ctx, AddRequest{Title: "Project task", Project: "Project", ProjectID: "project-id"}); err != nil || got.ID != "created-id" {
		t.Fatalf("add project task=%+v err=%v", got, err)
	}
	if got, err := client.AddProject(ctx, AddProjectRequest{Title: "Project"}); err != nil || got.Action != "add-project" {
		t.Fatalf("add-project=%+v err=%v", got, err)
	}
	title := "Renamed"
	if got, err := client.Update(ctx, UpdateRequest{ID: "existing-id", Title: &title}); err != nil || got.ID != "existing-id" {
		t.Fatalf("update=%+v err=%v", got, err)
	}
	if got, err := client.Move(ctx, MoveRequest{ID: "existing-id", Area: "Work", AreaID: "area-id"}); err != nil || got.Action != "move" {
		t.Fatalf("move=%+v err=%v", got, err)
	}
	if got, err := client.Detach(ctx, DetachRequest{ID: "existing-id", Project: true, Area: true}); err != nil || got.Action != "detach" {
		t.Fatalf("detach=%+v err=%v", got, err)
	}
	if _, err := client.Complete(ctx, "existing-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Cancel(ctx, "existing-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Show(ctx, "today"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(ctx, "quotes ' and unicode sữa"); err != nil {
		t.Fatal(err)
	}
	want := []string{"add", "add", "add-project", "update", "move", "detach", "complete", "cancel", "show", "search"}
	if !reflect.DeepEqual(runner.operations, want) {
		t.Fatalf("operations=%v want=%v", runner.operations, want)
	}
	request, ok := runner.requests[0].(AddRequest)
	if !ok || request.Title != "Mua sữa" {
		t.Fatalf("add request=%#v", runner.requests[0])
	}
	projectRequest, ok := runner.requests[1].(AddRequest)
	if !ok || projectRequest.Project != "Project" || projectRequest.ProjectID != "project-id" {
		t.Fatalf("project add request=%#v", runner.requests[1])
	}
	moveRequest, ok := runner.requests[4].(MoveRequest)
	if !ok || moveRequest.Area != "Work" || moveRequest.AreaID != "area-id" || moveRequest.ID != "existing-id" {
		t.Fatalf("move request=%#v", runner.requests[4])
	}
	detachRequest, ok := runner.requests[5].(DetachRequest)
	if !ok || !detachRequest.Project || !detachRequest.Area || detachRequest.ID != "existing-id" {
		t.Fatalf("detach request=%#v", runner.requests[5])
	}
}
