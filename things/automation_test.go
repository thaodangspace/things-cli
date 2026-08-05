package things

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

type fixtureRunner struct {
	response []byte
	err      error
	gotOp    string
	gotReq   any
}

func (r *fixtureRunner) Run(_ context.Context, operation string, request any) ([]byte, error) {
	r.gotOp = operation
	r.gotReq = request
	return r.response, r.err
}

func TestRunJSONDecodesSuccess(t *testing.T) {
	runner := &fixtureRunner{response: []byte(`{"ok":true,"data":{"operation":"health","request":{"text":"ok"}}}`)}
	var got AutomationHealth
	err := RunJSON(context.Background(), runner, "health", map[string]string{"text": "ok"}, &got)
	if err != nil {
		t.Fatal(err)
	}
	if runner.gotOp != "health" || got.Operation != "health" || got.Request["text"] != "ok" {
		t.Fatalf("runner=%+v response=%+v", runner, got)
	}
}

func TestRunJSONRejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response []byte
		want     string
	}{
		{name: "empty", response: nil, want: "empty response"},
		{name: "malformed", response: []byte("not json"), want: "decode JSON"},
		{name: "failed without detail", response: []byte(`{"ok":false}`), want: "without an error"},
		{name: "success without data", response: []byte(`{"ok":true}`), want: "no data"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := RunJSON(context.Background(), &fixtureRunner{response: tc.response}, tc.name, nil, &struct{}{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want substring %q", err, tc.want)
			}
			var ae *AutomationError
			if !errors.As(err, &ae) || ae.Kind != AutomationResponseFailure {
				t.Fatalf("error type=%T value=%v", err, err)
			}
		})
	}
}

func TestRunJSONReturnsScriptError(t *testing.T) {
	runner := &fixtureRunner{response: []byte(`{"ok":false,"error":{"code":"unsupported_operation","message":"not supported"}}`)}
	err := RunJSON(context.Background(), runner, "unknown", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("error=%v", err)
	}
	var ae *AutomationError
	if !errors.As(err, &ae) || ae.Kind != AutomationResponseFailure {
		t.Fatalf("error type=%T value=%v", err, err)
	}
}

func TestExecScriptPassesJSONAsArgument(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "runner.sh")
	script := "#!/bin/sh\ncat >/dev/null\nprintf '%s\\n%s\\n' \"$4\" \"$5\"\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	request := map[string]string{"text": "quote ' \" newline\n $HOME && echo unsafe"}
	raw, err := (ExecScript{Binary: scriptPath, Source: "ignored source"}).Run(context.Background(), "health", request)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != 2 || lines[0] != "health" {
		t.Fatalf("arguments=%q", lines)
	}
	var got map[string]string
	if err := json.Unmarshal([]byte(lines[1]), &got); err != nil {
		t.Fatalf("request argument is not JSON: %v; raw=%q", err, lines[1])
	}
	if got["text"] != request["text"] {
		t.Fatalf("request=%q, want %q", got["text"], request["text"])
	}
}

func TestExecScriptReportsProcessAndPermissionErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	for _, tc := range []struct {
		name       string
		script     string
		wantKind   AutomationFailureKind
		wantStderr string
	}{
		{name: "process", script: "#!/bin/sh\nprintf 'runner failed' >&2\nexit 7\n", wantKind: AutomationFailure, wantStderr: "runner failed"},
		{name: "permission", script: "#!/bin/sh\nprintf '(-1743) not authorized' >&2\nexit 1\n", wantKind: AutomationPermissionDenied, wantStderr: "not authorized"},
		{name: "missing application", script: "#!/bin/sh\nprintf '(-1728) Can’t get application' >&2\nexit 1\n", wantKind: AutomationApplicationMissing, wantStderr: "application"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scriptPath := filepath.Join(t.TempDir(), "runner.sh")
			if err := os.WriteFile(scriptPath, []byte(tc.script), 0o700); err != nil {
				t.Fatal(err)
			}
			_, err := (ExecScript{Binary: scriptPath}).Run(context.Background(), tc.name, nil)
			if err == nil {
				t.Fatal("expected process error")
			}
			var ae *AutomationError
			if !errors.As(err, &ae) || ae.Kind != tc.wantKind || !strings.Contains(ae.Stderr, tc.wantStderr) {
				t.Fatalf("error=%T %+v, want kind=%q stderr=%q", err, ae, tc.wantKind, tc.wantStderr)
			}
		})
	}
}

func TestExecScriptContextTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell")
	}
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "slow.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nsleep 2\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := (ExecScript{Binary: scriptPath, Source: ""}).Run(ctx, "health", nil)
	if err == nil {
		t.Fatal("expected timeout")
	}
	var ae *AutomationError
	if !errors.As(err, &ae) || ae.Kind != AutomationTimeout {
		t.Fatalf("error=%T %v", err, err)
	}
}

func TestExecScriptProjectTodoCreationContract(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("osascript is a macOS dependency")
	}

	const applicationLine = `var app = Application("com.culturedcode.ThingsMac");`
	if count := strings.Count(defaultAutomationScript, applicationLine); count != 1 {
		t.Fatalf("Things application construction count=%d, want 1", count)
	}
	source := strings.Replace(defaultAutomationScript, applicationLine,
		`var app = contractApplication("com.culturedcode.ThingsMac");`, 1) + `
var contractInserted = false;
var contractTask = {
  id: function () {
    if (!contractInserted) throw new Error("todo was not inserted into project");
    return "created-id";
  },
  show: function () {}
};
var contractProject = {
  exists: function () { return true; },
  id: function () { return "project-id"; },
  name: function () { return "vsee"; },
  toDos: {
    push: function (task) {
      if (task !== contractTask) throw new Error("unexpected todo inserted into project");
      contractInserted = true;
    }
  }
};
var missingProject = {
  exists: function () { return false; }
};
function contractApplication(identifier) {
  if (identifier !== "com.culturedcode.ThingsMac") throw new Error("unexpected application: " + identifier);
  return {
    projects: {
      byId: function (id) { return id === "project-id" ? contractProject : missingProject; },
      byName: function (name) { return name === "vsee" ? contractProject : missingProject; }
    },
    ToDo: function (properties) {
      if (!properties || properties.name !== "test2") throw new Error("unexpected todo properties");
      return contractTask;
    },
    make: function () { throw new Error("make must not be called for a project destination"); },
    move: function () { throw new Error("move must not be called for a project destination"); }
  };
}
`

	for _, request := range []AddRequest{
		{Title: "test2", Project: "vsee"},
		{Title: "test2", ProjectID: "project-id"},
	} {
		var result ActionResult
		if err := RunJSON(context.Background(), ExecScript{Source: source}, "add", request, &result); err != nil {
			t.Fatalf("add request=%+v: %v", request, err)
		}
		if result.Action != "add" || result.ID != "created-id" {
			t.Fatalf("result=%+v", result)
		}
	}

	var result ActionResult
	err := RunJSON(context.Background(), ExecScript{Source: source}, "add",
		AddRequest{Title: "test2", Project: "missing"}, &result)
	if err == nil || !strings.Contains(err.Error(), "project not found: missing") {
		t.Fatalf("error=%v, want project-not-found failure", err)
	}
}

func TestExecScriptQueryContract(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("osascript is a macOS dependency")
	}

	const applicationLine = `var app = Application("com.culturedcode.ThingsMac");`
	if count := strings.Count(defaultAutomationScript, applicationLine); count != 1 {
		t.Fatalf("Things application construction count=%d, want 1", count)
	}
	source := strings.Replace(defaultAutomationScript, applicationLine,
		`var app = contractApplication("com.culturedcode.ThingsMac");`, 1) + `
function fakeTask(id, title, notes, created, modified, deadline, start) {
  return {
    id: function () { return id; },
    properties: function () {
      return {
        pcls: "to-do",
        status: "open",
        name: title,
        notes: notes,
        creationDate: new Date(created),
        modificationDate: modified ? new Date(modified) : null,
        dueDate: deadline ? new Date(deadline) : null,
        activationDate: start ? new Date(start) : null
      };
    },
    tags: function () { return []; }
  };
}
var alpha = fakeTask("a", "Same", "Quarterly Review", "2026-08-01T00:00:00Z", "2026-08-03T00:00:00Z", "2026-08-07T00:00:00Z", "2026-08-02T00:00:00Z");
var beta = fakeTask("b", "same", "Other", "2026-08-01T00:00:00Z", "2026-08-01T00:00:00Z", null, null);
var gamma = fakeTask("c", "Alpha", "quarterly REVIEW", "2026-08-02T00:00:00Z", null, "2026-08-05T00:00:00Z", "2026-08-01T00:00:00Z");
var zeta = fakeTask("d", "Zeta", "Other", "2026-08-03T00:00:00Z", "2026-08-02T00:00:00Z", "2026-08-09T00:00:00Z", "2026-08-03T00:00:00Z");
var topTodos = [zeta, gamma, beta, alpha];
var inboxTodos = [alpha, zeta, gamma];
var inboxList = {
  id: function () { return "TMInboxListSource"; },
  name: function () { return "Inbox"; },
  exists: function () { return true; },
  toDos: function () { return inboxTodos; }
};
function missingList() { return { exists: function () { return false; } }; }
var lists = function () { return [inboxList]; };
lists.byId = function (id) { return id === "TMInboxListSource" ? inboxList : missingList(); };
lists.byName = function (name) { return name === "Inbox" ? inboxList : missingList(); };
function contractApplication(identifier) {
  if (identifier !== "com.culturedcode.ThingsMac") throw new Error("unexpected application: " + identifier);
  return {
    toDos: function () { return topTodos; },
    projects: function () { return []; },
    areas: function () { return []; },
    lists: lists
  };
}
`
	client := NewAutomationClient(ExecScript{Source: source})
	queryIDs := func(name string, filter Filter, want ...string) {
		t.Helper()
		items, err := client.Query(context.Background(), filter)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := make([]string, 0, len(items))
		for _, item := range items {
			got = append(got, item.ID)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: ids=%v, want %v", name, got, want)
		}
	}

	queryIDs("text", Filter{Text: "QUARTERLY", All: true}, "c", "a")
	queryIDs("created inclusive", Filter{CreatedAfter: "2026-08-01T00:00:00Z", CreatedBefore: "2026-08-01T00:00:00Z", All: true}, "a", "b")
	queryIDs("modified inclusive and null", Filter{ModifiedAfter: "2026-08-01T00:00:00Z", ModifiedBefore: "2026-08-01T00:00:00Z", All: true}, "b")
	queryIDs("deadline inclusive and null", Filter{DeadlineAfter: "2026-08-07", DeadlineBefore: "2026-08-07", All: true}, "a")
	queryIDs("start inclusive and null", Filter{StartAfter: "2026-08-01", StartBefore: "2026-08-01", All: true}, "c")
	queryIDs("composed before limit", Filter{Text: "quarterly", CreatedAfter: "2026-08-02", Sort: "title", Limit: 1}, "c")

	queryIDs("default limit and title order", Filter{Limit: 2}, "c", "a")
	for _, tc := range []struct {
		name string
		sort string
		want []string
	}{
		{"native", "native", []string{"d", "c", "b", "a"}},
		{"title", "title", []string{"c", "a", "b", "d"}},
		{"created", "created", []string{"a", "b", "c", "d"}},
		{"modified", "modified", []string{"b", "d", "a", "c"}},
		{"deadline", "deadline", []string{"c", "a", "d", "b"}},
		{"start", "start", []string{"c", "a", "d", "b"}},
	} {
		queryIDs(tc.name, Filter{Sort: tc.sort, All: true}, tc.want...)
	}
	queryIDs("reverse final sequence", Filter{Sort: "title", Reverse: true, All: true}, "d", "b", "a", "c")
	queryIDs("native list order", Filter{List: ListInbox, Sort: "native", All: true}, "a", "d", "c")
	queryIDs("all", Filter{All: true}, "c", "a", "b", "d")
}

func TestExecScriptHealthRoundTrip(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("osascript is a macOS dependency")
	}
	request := map[string]any{
		"text":  "quotes: ' \"; newline:\n; shell: $HOME && echo no; unicode: mua sữa",
		"items": []any{"one", 2, true, nil},
	}
	var got AutomationHealth
	if err := RunJSON(context.Background(), ExecScript{}, "health", request, &got); err != nil {
		t.Fatal(err)
	}
	if got.Operation != "health" {
		t.Fatalf("operation=%q", got.Operation)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(encoded, &want); err != nil {
		t.Fatal(err)
	}
	if !mapsEqual(got.Request, want) {
		t.Fatalf("request=%v, want %v", got.Request, want)
	}
}

func mapsEqual(a, b map[string]any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
