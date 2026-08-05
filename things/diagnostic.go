package things

import (
	"context"
	"errors"
	"fmt"
)

// DiagnosticStatus is the stable status reported for an environment check.
type DiagnosticStatus string

const (
	DiagnosticPass DiagnosticStatus = "pass"
	DiagnosticFail DiagnosticStatus = "fail"
	DiagnosticSkip DiagnosticStatus = "skip"
)

// DiagnosticCheck is intentionally limited to environment metadata. It never
// contains Things content.
type DiagnosticCheck struct {
	Name    string           `json:"name"`
	Status  DiagnosticStatus `json:"status"`
	Code    string           `json:"code,omitempty"`
	Message string           `json:"message"`
}

// DiagnosticReport is the result of the read-only doctor probe.
type DiagnosticReport struct {
	Healthy bool              `json:"healthy"`
	Version string            `json:"version"`
	Checks  []DiagnosticCheck `json:"checks"`
}

// DoctorEnvironment contains the local facts collected by the CLI. Keeping
// these facts as input makes diagnostics deterministic in unit tests and keeps
// OS/path probing out of the automation layer.
type DoctorEnvironment struct {
	GOOS                string
	GOARCH              string
	OsascriptPath       string
	OsascriptExists     bool
	OsascriptExecutable bool
}

// AutomationDiagnosis is the deliberately small response returned by the
// dedicated diagnose operation. It must not grow to include Things content.
type AutomationDiagnosis struct {
	Application      string `json:"application"`
	BundleIdentifier string `json:"bundle_identifier"`
}

type doctorHealth struct {
	Operation string         `json:"operation"`
	Request   map[string]any `json:"request"`
}

// Diagnose runs the read-only automation and protocol probes after checking
// the local platform and osascript facts. It returns a report even when checks
// fail. The runner is never used when macOS or osascript is unavailable.
func Diagnose(ctx context.Context, runner ScriptRunner, env DoctorEnvironment, version string) DiagnosticReport {
	if env.OsascriptPath == "" {
		env.OsascriptPath = "/usr/bin/osascript"
	}
	report := DiagnosticReport{Version: version, Checks: []DiagnosticCheck{}}

	platformOK := env.GOOS == "darwin"
	platformMessage := fmt.Sprintf("%s %s", env.GOOS, env.GOARCH)
	if platformOK {
		platformMessage = "macOS " + env.GOARCH
	}
	platformCheck := DiagnosticCheck{
		Name: "platform", Status: status(platformOK), Message: platformMessage,
	}
	if !platformOK {
		platformCheck.Code = "unsupported_platform"
	}
	report.Checks = append(report.Checks, platformCheck)

	osascriptOK := env.OsascriptExists && env.OsascriptExecutable
	osascriptMessage := env.OsascriptPath
	if !env.OsascriptExists {
		osascriptMessage = fmt.Sprintf("%s is missing; install macOS command-line tools or use a macOS host", env.OsascriptPath)
	} else if !env.OsascriptExecutable {
		osascriptMessage = fmt.Sprintf("%s exists but is not executable", env.OsascriptPath)
	}
	osascriptCheck := DiagnosticCheck{
		Name: "osascript", Status: status(osascriptOK), Message: osascriptMessage,
	}
	if !env.OsascriptExists {
		osascriptCheck.Code = "osascript_missing"
	} else if !env.OsascriptExecutable {
		osascriptCheck.Code = "osascript_not_executable"
	}
	report.Checks = append(report.Checks, osascriptCheck)

	if !platformOK {
		report.Checks = append(report.Checks,
			DiagnosticCheck{Name: "things-app", Status: DiagnosticSkip, Message: "not checked: Things automation requires macOS"},
			DiagnosticCheck{Name: "automation-permission", Status: DiagnosticSkip, Message: "not checked: Things automation requires macOS"},
			DiagnosticCheck{Name: "automation-protocol", Status: DiagnosticSkip, Message: "not checked: Things automation requires macOS and osascript"},
		)
	} else if !osascriptOK {
		report.Checks = append(report.Checks,
			DiagnosticCheck{Name: "things-app", Status: DiagnosticSkip, Message: "not checked: osascript is unavailable"},
			DiagnosticCheck{Name: "automation-permission", Status: DiagnosticSkip, Message: "not checked: osascript is unavailable"},
			DiagnosticCheck{Name: "automation-protocol", Status: DiagnosticSkip, Message: "not checked: osascript is unavailable"},
		)
	} else {
		protocolCheck := DiagnosticCheck{Name: "automation-protocol", Status: DiagnosticPass, Message: "JSON envelope round-trip succeeded"}
		var health doctorHealth
		healthErr := RunJSON(ctx, runner, "health", map[string]string{"probe": "things-cli-doctor"}, &health)
		if healthErr != nil || health.Operation != "health" || health.Request["probe"] != "things-cli-doctor" {
			protocolCheck.Status = DiagnosticFail
			protocolCheck.Code = diagnosticCode(healthErr, "protocol_failure")
			protocolCheck.Message = doctorFailureMessage("automation protocol", healthErr)
		}

		appCheck := DiagnosticCheck{Name: "things-app", Status: DiagnosticPass, Message: "Things 3 is available"}
		permissionCheck := DiagnosticCheck{Name: "automation-permission", Status: DiagnosticPass, Message: "Read-only automation succeeded"}
		var diagnosis AutomationDiagnosis
		err := RunJSON(ctx, runner, "diagnose", struct{}{}, &diagnosis)
		if err != nil {
			if isAutomationKind(err, AutomationApplicationMissing) {
				appCheck.Status = DiagnosticFail
				appCheck.Code = "application_missing"
				appCheck.Message = "Things 3 is not installed or could not be resolved; install or open Things 3"
				permissionCheck.Status = DiagnosticSkip
				permissionCheck.Message = "not checked: Things 3 is unavailable"
			} else if isAutomationKind(err, AutomationPermissionDenied) {
				permissionCheck.Status = DiagnosticFail
				permissionCheck.Code = "permission_denied"
				permissionCheck.Message = permissionRemediation
			} else {
				appCheck.Status = DiagnosticFail
				appCheck.Code = diagnosticCode(err, "application_probe_failed")
				appCheck.Message = doctorFailureMessage("Things application", err)
				permissionCheck.Status = DiagnosticSkip
				permissionCheck.Message = "not checked: Things application probe failed"
			}
		} else if diagnosis.Application == "" || diagnosis.BundleIdentifier == "" {
			appCheck.Status = DiagnosticFail
			appCheck.Code = "application_probe_invalid"
			appCheck.Message = "Things application probe returned incomplete metadata"
			permissionCheck.Status = DiagnosticSkip
			permissionCheck.Message = "not checked: Things application metadata is incomplete"
		} else if diagnosis.BundleIdentifier != "com.culturedcode.ThingsMac" {
			appCheck.Status = DiagnosticFail
			appCheck.Code = "application_identity_mismatch"
			appCheck.Message = "unexpected Things bundle identifier"
			permissionCheck.Status = DiagnosticSkip
			permissionCheck.Message = "not checked: Things application identity is invalid"
		}
		report.Checks = append(report.Checks, appCheck, permissionCheck, protocolCheck)
	}

	report.Checks = append(report.Checks, DiagnosticCheck{
		Name: "version", Status: DiagnosticPass, Message: version,
	})
	report.Healthy = true
	for _, check := range report.Checks {
		if check.Status == DiagnosticFail {
			report.Healthy = false
			break
		}
	}
	return report
}

const permissionRemediation = "Automation permission was denied; allow access for this terminal, agent host, launcher, or executable in System Settings → Privacy & Security → Automation"

func status(ok bool) DiagnosticStatus {
	if ok {
		return DiagnosticPass
	}
	return DiagnosticFail
}

func isAutomationKind(err error, kind AutomationFailureKind) bool {
	var automationErr *AutomationError
	return errors.As(err, &automationErr) && automationErr.Kind == kind
}

func diagnosticCode(err error, fallback string) string {
	var automationErr *AutomationError
	if errors.As(err, &automationErr) {
		switch automationErr.Kind {
		case AutomationTimeout:
			return "timeout"
		case AutomationCanceled:
			return "canceled"
		case AutomationPermissionDenied:
			return "permission_denied"
		case AutomationApplicationMissing:
			return "application_missing"
		}
	}
	return fallback
}

func doctorFailureMessage(subject string, err error) string {
	if err == nil {
		return subject + " probe returned an invalid {ok,data} envelope"
	}
	var automationErr *AutomationError
	if errors.As(err, &automationErr) {
		switch automationErr.Kind {
		case AutomationTimeout:
			return subject + " probe timed out; increase --timeout or check that Things is responding"
		case AutomationPermissionDenied:
			return permissionRemediation
		case AutomationApplicationMissing:
			return "Things 3 is not installed or could not be resolved; install or open Things 3"
		case AutomationResponseFailure:
			return subject + " probe returned an invalid {ok,data} envelope"
		}
	}
	return subject + " probe failed"
}
