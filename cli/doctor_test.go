package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/thaodangspace/things-cli/things"
)

type doctorFixtureRunner struct{}

func (doctorFixtureRunner) Run(_ context.Context, operation string, _ any) ([]byte, error) {
	switch operation {
	case "health":
		return []byte(`{"ok":true,"data":{"operation":"health","request":{"probe":"things-cli-doctor"}}}`), nil
	case "diagnose":
		return []byte(`{"ok":true,"data":{"application":"Things 3","bundle_identifier":"com.culturedcode.ThingsMac"}}`), nil
	default:
		return nil, os.ErrNotExist
	}
}

func TestDoctorCommandJSONHealthy(t *testing.T) {
	oldGOOS, oldGOARCH, oldStat, oldRunner := doctorGOOS, doctorGOARCH, doctorStat, doctorRunner
	t.Cleanup(func() { doctorGOOS, doctorGOARCH, doctorStat, doctorRunner = oldGOOS, oldGOARCH, oldStat, oldRunner })
	tmp := t.TempDir() + "/osascript"
	if err := os.WriteFile(tmp, []byte("runner"), 0o755); err != nil {
		t.Fatal(err)
	}
	doctorGOOS, doctorGOARCH = "darwin", "arm64"
	doctorStat = func(string) (os.FileInfo, error) { return os.Stat(tmp) }
	doctorRunner = func(string) things.ScriptRunner { return doctorFixtureRunner{} }

	root := newRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"doctor", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"healthy": true`) || !strings.Contains(out.String(), `"name": "automation-permission"`) {
		t.Fatalf("output=%s", out.String())
	}
}

func TestDoctorCommandReportsFailureAndExitOne(t *testing.T) {
	oldGOOS, oldGOARCH, oldStat, oldRunner := doctorGOOS, doctorGOARCH, doctorStat, doctorRunner
	t.Cleanup(func() { doctorGOOS, doctorGOARCH, doctorStat, doctorRunner = oldGOOS, oldGOARCH, oldStat, oldRunner })
	doctorGOOS, doctorGOARCH = "linux", "amd64"
	doctorStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	doctorRunner = func(string) things.ScriptRunner { t.Fatal("runner used for unsupported platform"); return nil }

	root := newRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"doctor", "--human"})
	err := root.Execute()
	if err == nil || exitCodeFor(err) != exitRuntime {
		t.Fatalf("err=%v exit=%d", err, exitCodeFor(err))
	}
	for _, name := range []string{"platform", "osascript", "things-app", "automation-permission", "automation-protocol", "version"} {
		if !strings.Contains(out.String(), name) {
			t.Fatalf("missing %q in output=%s", name, out.String())
		}
	}
}
