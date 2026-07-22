package things

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
