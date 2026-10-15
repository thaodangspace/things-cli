package cli

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

const doctorOsascriptPath = "/usr/bin/osascript"

// These variables keep host probing injectable for deterministic tests.
var (
	doctorGOOS   = runtime.GOOS
	doctorGOARCH = runtime.GOARCH
	doctorStat   = os.Stat
	doctorRunner = func(path string) things.ScriptRunner {
		return things.ExecScript{
			Binary: path,
			Logger: func(operation string, duration time.Duration, err error) {
				if !opts.verbose {
					return
				}
				state := "ok"
				if err != nil {
					state = "error"
				}
				fmt.Fprintf(os.Stderr, "things automation operation=%s duration=%s status=%s\\n", operation, duration.Round(time.Millisecond), state)
			},
		}
	}
)

type doctorFailure struct{}

func (doctorFailure) Error() string { return "doctor checks failed" }

func newDoctorCommand() *cobra.Command {
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose macOS, Things, and Automation readiness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			env := things.DoctorEnvironment{
				GOOS:          doctorGOOS,
				GOARCH:        doctorGOARCH,
				OsascriptPath: doctorOsascriptPath,
			}
			if info, err := doctorStat(doctorOsascriptPath); err == nil {
				env.OsascriptExists = true
				env.OsascriptExecutable = info.Mode()&0111 != 0
			}

			var runner things.ScriptRunner
			if env.GOOS == "darwin" && env.OsascriptExists && env.OsascriptExecutable {
				runner = doctorRunner(env.OsascriptPath)
			}
			ctx, cancel := withTimeout(cmd)
			defer cancel()
			report := things.Diagnose(ctx, runner, env, version)
			human := opts.human && !jsonOutput
			if err := writeDiagnostic(cmd, report, human); err != nil {
				return err
			}
			if !report.Healthy {
				return doctorFailure{}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print JSON output (the default)")
	return cmd
}

type diagnosticEnvelope struct {
	OK    bool                     `json:"ok"`
	Data  things.DiagnosticReport  `json:"data"`
	Error *diagnosticEnvelopeError `json:"error,omitempty"`
}

type diagnosticEnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeDiagnostic(cmd *cobra.Command, report things.DiagnosticReport, human bool) error {
	if human {
		state := "healthy"
		if !report.Healthy {
			state = "unhealthy"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "things-cli doctor: %s (version %s)\n", state, report.Version); err != nil {
			return err
		}
		for _, check := range report.Checks {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s: %s\n", check.Status, check.Name, check.Message); err != nil {
				return err
			}
		}
		return nil
	}
	envelope := diagnosticEnvelope{OK: report.Healthy, Data: report}
	if !report.Healthy {
		envelope.Error = &diagnosticEnvelopeError{Code: "doctor_failed", Message: "one or more doctor checks failed"}
	}
	return writeJSON(cmd.OutOrStdout(), envelope)
}
