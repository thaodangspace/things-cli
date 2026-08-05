package things

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type diagnosticRunner struct {
	responses  map[string][]byte
	operations []string
}

func (r *diagnosticRunner) Run(_ context.Context, operation string, _ any) ([]byte, error) {
	r.operations = append(r.operations, operation)
	if response, ok := r.responses[operation]; ok {
		return response, nil
	}
	return nil, errors.New("unexpected operation")
}

func diagnosticResponse(data string) []byte {
	return []byte(`{"ok":true,"data":` + data + `}`)
}

func doctorTestEnvironment() DoctorEnvironment {
	return DoctorEnvironment{GOOS: "darwin", GOARCH: "arm64", OsascriptPath: "/usr/bin/osascript", OsascriptExists: true, OsascriptExecutable: true}
}

func TestDiagnoseHealthy(t *testing.T) {
	runner := &diagnosticRunner{responses: map[string][]byte{
		"health":   diagnosticResponse(`{"operation":"health","request":{"probe":"things-cli-doctor"}}`),
		"diagnose": diagnosticResponse(`{"application":"Things 3","bundle_identifier":"com.culturedcode.ThingsMac"}`),
	}}
	report := Diagnose(context.Background(), runner, doctorTestEnvironment(), "1.2.3")
	if !report.Healthy || report.Version != "1.2.3" {
		t.Fatalf("report=%+v", report)
	}
	want := []string{"health", "diagnose"}
	if !reflect.DeepEqual(runner.operations, want) {
		t.Fatalf("operations=%v want=%v", runner.operations, want)
	}
	for _, check := range report.Checks {
		if check.Status != DiagnosticPass {
			t.Fatalf("check=%+v", check)
		}
	}
}

func TestDiagnoseReportsPrerequisiteFailures(t *testing.T) {
	runner := &diagnosticRunner{}
	report := Diagnose(context.Background(), runner, DoctorEnvironment{GOOS: "linux", GOARCH: "amd64", OsascriptPath: "/usr/bin/osascript"}, "dev")
	if report.Healthy {
		t.Fatal("non-macOS report unexpectedly healthy")
	}
	if len(runner.operations) != 0 {
		t.Fatalf("automation operations=%v", runner.operations)
	}
	for _, check := range report.Checks {
		if check.Name == "things-app" && check.Status != DiagnosticSkip {
			t.Fatalf("things app check=%+v", check)
		}
	}
}

func TestDiagnoseClassifiesPermissionDenial(t *testing.T) {
	runner := &diagnosticRunner{responses: map[string][]byte{
		"health":   diagnosticResponse(`{"operation":"health","request":{"probe":"things-cli-doctor"}}`),
		"diagnose": []byte(`{"ok":false,"error":{"code":"permission_denied","message":"permission denied"}}`),
	}}
	report := Diagnose(context.Background(), runner, doctorTestEnvironment(), "dev")
	if report.Healthy {
		t.Fatal("permission-denied report unexpectedly healthy")
	}
	for _, check := range report.Checks {
		if check.Name == "automation-permission" {
			if check.Status != DiagnosticFail || check.Message == "" {
				t.Fatalf("permission check=%+v", check)
			}
			if !containsText(check.Message, "Privacy & Security") {
				t.Fatalf("permission remediation=%q", check.Message)
			}
		}
	}
}

func TestDiagnoseClassifiesApplicationAndProtocolFailures(t *testing.T) {
	runner := &diagnosticRunner{responses: map[string][]byte{
		"health":   []byte("not json"),
		"diagnose": []byte(`{"ok":false,"error":{"code":"application_missing","message":"missing"}}`),
	}}
	report := Diagnose(context.Background(), runner, doctorTestEnvironment(), "dev")
	if report.Healthy {
		t.Fatal("failed report unexpectedly healthy")
	}
	statuses := make(map[string]DiagnosticStatus)
	for _, check := range report.Checks {
		statuses[check.Name] = check.Status
	}
	if statuses["things-app"] != DiagnosticFail || statuses["automation-protocol"] != DiagnosticFail {
		t.Fatalf("statuses=%v", statuses)
	}
}

func TestDiagnoseReportsMissingOsascriptAndKeepsAllChecks(t *testing.T) {
	runner := &diagnosticRunner{}
	report := Diagnose(context.Background(), runner, DoctorEnvironment{
		GOOS: "darwin", GOARCH: "arm64", OsascriptPath: "/usr/bin/osascript",
	}, "dev")
	if report.Healthy || len(report.Checks) != 6 {
		t.Fatalf("report=%+v", report)
	}
	for _, check := range report.Checks {
		if check.Name == "osascript" && check.Status != DiagnosticFail {
			t.Fatalf("osascript check=%+v", check)
		}
	}
}

func containsText(value, want string) bool {
	for i := 0; i+len(want) <= len(value); i++ {
		if value[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
