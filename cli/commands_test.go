package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/thaodangspace/things-cli/things"
)

type fixtureService struct{}

func fixtureRef(id, title string) *things.Ref { return &things.Ref{ID: id, Title: title} }

func fixtureItems() []things.Item {
	return []things.Item{
		{ID: "proj-alpha", Type: "project", Status: "open", Title: "Alpha Project", Start: "anytime", Area: fixtureRef("area-work", "Work"), Tags: []string{"work"}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-done", Type: "to-do", Status: "completed", Title: "Done Task", Start: "anytime", Area: fixtureRef("area-work", "Work"), Tags: []string{}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-canceled", Type: "to-do", Status: "canceled", Title: "Canceled Task", Start: "anytime", Area: fixtureRef("area-work", "Work"), Tags: []string{}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-today", Type: "to-do", Status: "open", Title: "Today Task", Start: "unknown", Area: fixtureRef("area-work", "Work"), Project: fixtureRef("proj-alpha", "Alpha Project"), Tags: []string{"urgent"}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-upcoming", Type: "to-do", Status: "open", Title: "Upcoming Task", Start: "unknown", Area: fixtureRef("area-work", "Work"), Project: fixtureRef("proj-alpha", "Alpha Project"), Tags: []string{}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-inbox", Type: "to-do", Status: "open", Title: "Inbox Task", Start: "inbox", Tags: []string{}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-anytime", Type: "to-do", Status: "open", Title: "Anytime Task", Start: "anytime", Tags: []string{}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-someday", Type: "to-do", Status: "open", Title: "Someday Task", Start: "someday", Tags: []string{}, Checklist: []things.ChecklistItem{}},
		{ID: "todo-trash", Type: "to-do", Status: "canceled", Title: "Trash Task", Start: "unknown", Trashed: true, Tags: []string{}, Checklist: []things.ChecklistItem{}},
	}
}

func (fixtureService) List(_ context.Context, list things.ListKind, limit int) ([]things.Item, error) {
	byList := map[things.ListKind][]string{
		things.ListInbox:    {"todo-inbox"},
		things.ListToday:    {"proj-alpha", "todo-today"},
		things.ListUpcoming: {"proj-alpha", "todo-upcoming"},
		things.ListAnytime:  {"todo-anytime", "proj-alpha", "todo-today", "todo-upcoming"},
		things.ListSomeday:  {"todo-someday"},
		things.ListLogbook:  {"todo-done", "todo-canceled"},
		things.ListTrash:    {"todo-trash"},
	}
	return fixtureSelect(byList[list], limit), nil
}

func fixtureSelect(ids []string, limit int) []things.Item {
	all := fixtureItems()
	byID := make(map[string]things.Item, len(all))
	for _, item := range all {
		byID[item.ID] = item
	}
	out := make([]things.Item, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			out = append(out, item)
		}
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (fixtureService) Query(_ context.Context, filter things.Filter) ([]things.Item, error) {
	items := fixtureItems()
	if filter.List != things.ListNone {
		listItems, _ := (fixtureService{}).List(context.Background(), filter.List, 100)
		items = listItems
	}
	out := make([]things.Item, 0, len(items))
	for _, item := range items {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Type != "" && item.Type != filter.Type && !(filter.Type == "todo" && item.Type == "to-do") {
			continue
		}
		if filter.Tag != "" && !contains(item.Tags, filter.Tag) {
			continue
		}
		if filter.Area != "" && (item.Area == nil || (item.Area.ID != filter.Area && item.Area.Title != filter.Area)) {
			continue
		}
		if filter.Project != "" && (item.Project == nil || (item.Project.ID != filter.Project && item.Project.Title != filter.Project)) {
			continue
		}
		out = append(out, item)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (fixtureService) Get(_ context.Context, id string) (things.Item, error) {
	for _, item := range fixtureItems() {
		if item.ID == id {
			return item, nil
		}
	}
	return things.Item{}, things.ErrNotFound
}

func (fixtureService) ListProjects(_ context.Context, area string, limit int) ([]things.Item, error) {
	items := []string{"proj-alpha"}
	if area != "" && area != "Work" && area != "area-work" {
		items = nil
	}
	return fixtureSelect(items, limit), nil
}

func (fixtureService) ListAreas(context.Context) ([]things.Area, error) {
	return []things.Area{{ID: "area-work", Title: "Work"}}, nil
}

func (fixtureService) ListTags(context.Context) ([]things.Tag, error) {
	return []things.Tag{{ID: "tag-work", Title: "work"}}, nil
}

func (fixtureService) Add(context.Context, things.AddRequest) (things.ActionResult, error) {
	return things.ActionResult{Action: "add", ID: "todo-today"}, nil
}

type recordingService struct {
	fixtureService
	addRequest things.AddRequest
}

func (s *recordingService) Add(_ context.Context, request things.AddRequest) (things.ActionResult, error) {
	s.addRequest = request
	return things.ActionResult{Action: "add", ID: "todo-today"}, nil
}

func (fixtureService) AddProject(context.Context, things.AddProjectRequest) (things.ActionResult, error) {
	return things.ActionResult{Action: "add-project", ID: "proj-alpha"}, nil
}
func (fixtureService) Update(context.Context, things.UpdateRequest) (things.ActionResult, error) {
	return things.ActionResult{Action: "update", ID: "todo-today"}, nil
}
func (fixtureService) Move(context.Context, things.MoveRequest) (things.ActionResult, error) {
	return things.ActionResult{Action: "move", ID: "todo-today"}, nil
}
func (fixtureService) Detach(context.Context, things.DetachRequest) (things.ActionResult, error) {
	return things.ActionResult{Action: "detach", ID: "todo-today"}, nil
}
func (fixtureService) Delete(context.Context, string) (things.ActionResult, error) {
	return things.ActionResult{Action: "delete", ID: "todo-today"}, nil
}
func (fixtureService) EmptyTrash(context.Context) (things.ActionResult, error) {
	return things.ActionResult{Action: "empty-trash"}, nil
}
func (fixtureService) Complete(context.Context, string) (things.ActionResult, error) {
	return things.ActionResult{Action: "complete", ID: "todo-today"}, nil
}
func (fixtureService) Cancel(context.Context, string) (things.ActionResult, error) {
	return things.ActionResult{Action: "cancel", ID: "todo-today"}, nil
}
func (fixtureService) Show(context.Context, string) (things.ActionResult, error) {
	return things.ActionResult{Action: "show", ID: "today"}, nil
}
func (fixtureService) Search(context.Context, string) (things.ActionResult, error) {
	return things.ActionResult{Action: "search"}, nil
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	oldService := thingsService
	thingsService = fixtureService{}
	t.Cleanup(func() { thingsService = oldService })
	root := newRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func decodeData(t *testing.T, stdout string, v any) {
	t.Helper()
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if !env.OK {
		t.Fatalf("ok=false: %s", stdout)
	}
	if err := json.Unmarshal(env.Data, v); err != nil {
		t.Fatalf("decode data: %v", err)
	}
}

func TestQueryCommand(t *testing.T) {
	out, err := runCLI(t, "query", "--tag", "urgent")
	if err != nil {
		t.Fatal(err)
	}
	var items []struct{ ID string }
	decodeData(t, out, &items)
	if len(items) != 1 || items[0].ID != "todo-today" {
		t.Fatalf("items = %+v", items)
	}
}

func TestEmptyQueryJSONDataIsArray(t *testing.T) {
	out, err := runCLI(t, "query", "--tag", "missing")
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if !env.OK || strings.TrimSpace(string(env.Data)) != "[]" {
		t.Fatalf("out = %s, want data: []", out)
	}
}

func TestGetCommandHuman(t *testing.T) {
	out, err := runCLI(t, "get", "todo-today", "--human")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "todo-today Today Task [to-do open]") {
		t.Fatalf("out = %q", out)
	}
}

func TestBuiltInLists(t *testing.T) {
	for _, tc := range []struct{ cmd, id string }{{"inbox", "todo-inbox"}, {"today", "todo-today"}, {"upcoming", "todo-upcoming"}, {"someday", "todo-someday"}, {"trash", "todo-trash"}} {
		t.Run(tc.cmd, func(t *testing.T) {
			out, err := runCLI(t, tc.cmd)
			if err != nil {
				t.Fatal(err)
			}
			var items []struct{ ID string }
			decodeData(t, out, &items)
			found := false
			for _, it := range items {
				if it.ID == tc.id {
					found = true
				}
			}
			if !found {
				t.Fatalf("%s not found in %+v", tc.id, items)
			}
		})
	}
}

func TestListAreasTagsProjects(t *testing.T) {
	out, err := runCLI(t, "list-areas")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "area-work") {
		t.Fatalf("areas out = %s", out)
	}
	out, err = runCLI(t, "list-tags", "--human")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "tag-work work") {
		t.Fatalf("tags out = %s", out)
	}
	out, err = runCLI(t, "list-projects", "--area", "Work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "proj-alpha") {
		t.Fatalf("projects out = %s", out)
	}
}

func TestAddCommandUsesSynchronousAutomation(t *testing.T) {
	out, err := runCLI(t, "add", "--title", "Today Task", "--notes", "a&b", "--when", "today", "--tags", "work,urgent", "--wait")
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Action string
		ID     string
	}
	decodeData(t, out, &res)
	if res.Action != "add" || res.ID != "todo-today" {
		t.Fatalf("result = %+v", res)
	}
	if strings.Contains(out, "url") {
		t.Fatalf("automation result unexpectedly contains URL: %s", out)
	}
}

func TestAddCommandSupportsProjectDestination(t *testing.T) {
	service := &recordingService{}
	oldService := thingsService
	thingsService = service
	t.Cleanup(func() { thingsService = oldService })
	root := newRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"add", "--title", "Project Task", "--project", "Alpha Project", "--project-id", "proj-alpha"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if service.addRequest.Project != "Alpha Project" || service.addRequest.ProjectID != "proj-alpha" {
		t.Fatalf("add request=%+v", service.addRequest)
	}
}

func TestUpdateDoesNotRequireToken(t *testing.T) {
	out, err := runCLI(t, "update", "todo-today", "--notes", "", "--completed")
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Action string
		ID     string
	}
	decodeData(t, out, &res)
	if res.Action != "update" || res.ID != "todo-today" {
		t.Fatalf("result = %+v", res)
	}
}

func TestCompleteCancelShowSearch(t *testing.T) {
	oldService := thingsService
	thingsService = fixtureService{}
	t.Cleanup(func() { thingsService = oldService })
	for _, args := range [][]string{{"complete", "todo-today"}, {"cancel", "todo-today"}, {"show", "today"}, {"search", "ship it"}} {
		root := newRootCommand()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !strings.Contains(out.String(), `"ok": true`) {
			t.Fatalf("%v output = %s", args, out.String())
		}
	}
}

type recordingModifyService struct {
	fixtureService
	moveRequest   things.MoveRequest
	detachRequest things.DetachRequest
	deleteID      string
	emptyTrash    int
	showTargets   []string
}

func (s *recordingModifyService) Move(_ context.Context, request things.MoveRequest) (things.ActionResult, error) {
	s.moveRequest = request
	return things.ActionResult{Action: "move", ID: request.ID}, nil
}

func (s *recordingModifyService) Detach(_ context.Context, request things.DetachRequest) (things.ActionResult, error) {
	s.detachRequest = request
	return things.ActionResult{Action: "detach", ID: request.ID}, nil
}

func (s *recordingModifyService) Delete(_ context.Context, id string) (things.ActionResult, error) {
	s.deleteID = id
	return things.ActionResult{Action: "delete", ID: id}, nil
}

func (s *recordingModifyService) EmptyTrash(context.Context) (things.ActionResult, error) {
	s.emptyTrash++
	return things.ActionResult{Action: "empty-trash"}, nil
}

func (s *recordingModifyService) Show(_ context.Context, target string) (things.ActionResult, error) {
	s.showTargets = append(s.showTargets, target)
	return things.ActionResult{Action: "show", ID: target}, nil
}

func TestMoveCommandDestinationVariants(t *testing.T) {
	oldService := thingsService
	service := &recordingModifyService{}
	thingsService = service
	t.Cleanup(func() { thingsService = oldService })

	for _, tc := range []struct {
		args []string
		want things.MoveRequest
	}{
		{args: []string{"move", "todo-id", "--list", "Anytime"}, want: things.MoveRequest{ID: "todo-id", List: "Anytime"}},
		{args: []string{"move", "todo-id", "--project-id", "proj-alpha"}, want: things.MoveRequest{ID: "todo-id", ProjectID: "proj-alpha"}},
		{args: []string{"move", "todo-id", "--project", "Launch"}, want: things.MoveRequest{ID: "todo-id", Project: "Launch"}},
		{args: []string{"move", "todo-id", "--area", "Work"}, want: things.MoveRequest{ID: "todo-id", Area: "Work"}},
		{args: []string{"move", "todo-id", "--area-id", "area-work"}, want: things.MoveRequest{ID: "todo-id", AreaID: "area-work"}},
	} {
		root := newRootCommand()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(tc.args)
		if err := root.Execute(); err != nil {
			t.Fatalf("args=%v: %v", tc.args, err)
		}
		if service.moveRequest != tc.want {
			t.Fatalf("args=%v moveRequest=%+v want=%+v", tc.args, service.moveRequest, tc.want)
		}
	}
}

func TestMoveOutputFormats(t *testing.T) {
	out, err := runCLI(t, "move", "todo-id", "--area", "Work")
	if err != nil {
		t.Fatal(err)
	}
	var result things.ActionResult
	decodeData(t, out, &result)
	if result.Action != "move" || result.ID != "todo-today" {
		t.Fatalf("result=%+v", result)
	}

	out, err = runCLI(t, "detach", "todo-id", "--area", "--human")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "detach todo-today" {
		t.Fatalf("human output=%q", out)
	}
}

func TestDetachCommandScopes(t *testing.T) {
	oldService := thingsService
	service := &recordingModifyService{}
	thingsService = service
	t.Cleanup(func() { thingsService = oldService })

	for _, tc := range []struct {
		args []string
		want things.DetachRequest
	}{
		{args: []string{"detach", "todo-id", "--project"}, want: things.DetachRequest{ID: "todo-id", Project: true}},
		{args: []string{"detach", "todo-id", "--area"}, want: things.DetachRequest{ID: "todo-id", Area: true}},
		{args: []string{"detach", "todo-id", "--all"}, want: things.DetachRequest{ID: "todo-id", Project: true, Area: true, All: true}},
	} {
		root := newRootCommand()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(tc.args)
		if err := root.Execute(); err != nil {
			t.Fatalf("args=%v: %v", tc.args, err)
		}
		if service.detachRequest != tc.want {
			t.Fatalf("args=%v detachRequest=%+v want=%+v", tc.args, service.detachRequest, tc.want)
		}
	}
}

func TestDeleteAndEmptyTrashCommands(t *testing.T) {
	oldService := thingsService
	service := &recordingModifyService{}
	thingsService = service
	t.Cleanup(func() { thingsService = oldService })

	root := newRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"delete", "todo-id", "--reveal", "--human"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "delete todo-id" || service.deleteID != "todo-id" {
		t.Fatalf("delete output=%q service=%+v", out.String(), service)
	}
	if len(service.showTargets) != 1 || service.showTargets[0] != "trash" {
		t.Fatalf("show targets=%v", service.showTargets)
	}

	out.Reset()
	root = newRootCommand()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"empty-trash", "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result things.ActionResult
	decodeData(t, out.String(), &result)
	if result.Action != "empty-trash" || service.emptyTrash != 1 {
		t.Fatalf("empty-trash result=%+v calls=%d", result, service.emptyTrash)
	}
}

func TestEmptyTrashRequiresConfirmationWithoutAutomation(t *testing.T) {
	oldService := thingsService
	service := &recordingModifyService{}
	thingsService = service
	t.Cleanup(func() { thingsService = oldService })
	_, err := runCLI(t, "empty-trash")
	if exitCodeFor(err) != exitUsage {
		t.Fatalf("exit=%d err=%v", exitCodeFor(err), err)
	}
	if service.emptyTrash != 0 {
		t.Fatalf("empty-trash automation calls=%d", service.emptyTrash)
	}
}

func TestMoveDetachUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"move", "todo-id"}, // no destination
		{"move", "todo-id", "--list", "Today", "--area", "Work"}, // multiple destinations
		{"move", "todo-id", "--project", "P", "--area", "A"},     // multiple destinations
		{"move", "todo-id", "--list", "Upcoming"},                // upcoming unsupported
		{"move"}, // missing id
		{"detach", "todo-id", "--project", "--area"}, // exclusive
		{"detach", "todo-id", "--all", "--project"},  // all exclusive
		{"detach", "todo-id"},                        // no scope
	} {
		_, err := runCLI(t, args...)
		if exitCodeFor(err) != exitUsage {
			t.Fatalf("args=%v exit=%d err=%v", args, exitCodeFor(err), err)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	_, err := runCLI(t, "query", "--limit", "0")
	if exitCodeFor(err) != exitUsage {
		t.Fatalf("limit exit = %d", exitCodeFor(err))
	}
	for _, args := range [][]string{
		{"add", "--title", "Task", "--checklist-items", "one"},
		{"add", "--title", "Task", "--list", "Today", "--project", "Project"},
		{"update", "id", "--heading", "Heading"},
		{"add", "--title", "Task", "--when", "evening"},
	} {
		_, err := runCLI(t, args...)
		if exitCodeFor(err) != exitUsage {
			t.Fatalf("args=%v exit=%d err=%v", args, exitCodeFor(err), err)
		}
	}
	root := newRootCommand()
	root.SetArgs([]string{"bogus"})
	if got := exitCodeFor(root.Execute()); got != exitUsage {
		t.Fatalf("unknown command exit = %d", got)
	}
}
