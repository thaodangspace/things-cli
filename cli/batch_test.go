package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/thaodangspace/things-cli/things"
)

type batchProbeService struct {
	fixtureService
	calls    []string
	failCall string
}

func (s *batchProbeService) Add(context.Context, things.AddRequest) (things.ActionResult, error) {
	s.calls = append(s.calls, "add")
	if s.failCall == "add" {
		return things.ActionResult{}, errors.New("add failed")
	}
	return things.ActionResult{Action: "add", ID: "new-task"}, nil
}

func (s *batchProbeService) Update(context.Context, things.UpdateRequest) (things.ActionResult, error) {
	s.calls = append(s.calls, "update")
	if s.failCall == "update" {
		return things.ActionResult{}, errors.New("update failed")
	}
	return things.ActionResult{Action: "update", ID: "task"}, nil
}

func (s *batchProbeService) Complete(context.Context, string) (things.ActionResult, error) {
	s.calls = append(s.calls, "complete")
	if s.failCall == "complete" {
		return things.ActionResult{}, errors.New("complete failed")
	}
	return things.ActionResult{Action: "complete", ID: "task"}, nil
}

func (s *batchProbeService) Cancel(context.Context, string) (things.ActionResult, error) {
	s.calls = append(s.calls, "cancel")
	if s.failCall == "cancel" {
		return things.ActionResult{}, errors.New("cancel failed")
	}
	return things.ActionResult{Action: "cancel", ID: "task"}, nil
}

func runBatchTest(t *testing.T, service things.Service, input string, args ...string) (string, error) {
	t.Helper()
	oldService := thingsService
	thingsService = service
	t.Cleanup(func() { thingsService = oldService })
	root := newRootCommand()
	var out bytes.Buffer
	root.SetIn(strings.NewReader(input))
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"batch"}, args...))
	err := root.Execute()
	return out.String(), err
}

func decodeBatchResults(t *testing.T, output string) []batchResult {
	t.Helper()
	var results []batchResult
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		var result batchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			t.Fatalf("invalid batch JSON %q: %v", line, err)
		}
		results = append(results, result)
	}
	return results
}

func TestBatchExecutesSequentiallyAndCorrelatesClientID(t *testing.T) {
	service := &batchProbeService{}
	output, err := runBatchTest(t, service, `{"client_id":"first","operation":"add","request":{"title":"Task"}}
{"operation":"complete","request":{"id":"task"}}
`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(service.calls, ",") != "add,complete" {
		t.Fatalf("calls=%v", service.calls)
	}
	results := decodeBatchResults(t, output)
	if len(results) != 2 || !results[0].OK || results[0].ClientID == nil || *results[0].ClientID != "first" || results[1].ClientID != nil {
		t.Fatalf("results=%+v", results)
	}
}

func TestBatchStopsOrContinuesAfterRuntimeFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "stop", want: "complete"},
		{name: "continue", args: []string{"--continue-on-error"}, want: "complete,cancel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &batchProbeService{failCall: "complete"}
			output, err := runBatchTest(t, service, `{"operation":"complete","request":{"id":"task"}}
{"operation":"cancel","request":{"id":"task"}}
`, tc.args...)
			if exitCodeFor(err) != exitRuntime {
				t.Fatalf("exit=%d err=%v", exitCodeFor(err), err)
			}
			if strings.Join(service.calls, ",") != tc.want {
				t.Fatalf("calls=%v want=%s", service.calls, tc.want)
			}
			results := decodeBatchResults(t, output)
			if len(results) != len(strings.Split(tc.want, ",")) || results[0].OK {
				t.Fatalf("results=%+v", results)
			}
		})
	}
}

func TestBatchValidateOnlyDoesNotCallService(t *testing.T) {
	service := &batchProbeService{}
	output, err := runBatchTest(t, service, "\n"+`{"operation":"add","request":{"title":"  Task  "}}`+"\n", "--validate-only")
	if err != nil {
		t.Fatal(err)
	}
	if len(service.calls) != 0 {
		t.Fatalf("service calls=%v", service.calls)
	}
	results := decodeBatchResults(t, output)
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("results=%+v", results)
	}
	data, ok := results[0].Data.(map[string]any)
	if !ok || data["operation"] != "add" {
		t.Fatalf("normalized data=%v", results[0].Data)
	}
}

func TestBatchMalformedJSONIncludesLineAndUsageExit(t *testing.T) {
	output, err := runBatchTest(t, &batchProbeService{}, "\nnot-json\n")
	if exitCodeFor(err) != exitUsage {
		t.Fatalf("exit=%d err=%v", exitCodeFor(err), err)
	}
	results := decodeBatchResults(t, output)
	if len(results) != 1 || results[0].OK || results[0].Error == nil || results[0].Error.Code != "invalid_json" || !strings.Contains(results[0].Error.Message, "line 2") {
		t.Fatalf("results=%+v", results)
	}
}

func TestBatchRejectsHumanOutput(t *testing.T) {
	_, err := runBatchTest(t, &batchProbeService{}, "", "--human")
	if exitCodeFor(err) != exitUsage {
		t.Fatalf("exit=%d err=%v", exitCodeFor(err), err)
	}
}
