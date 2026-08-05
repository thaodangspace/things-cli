package things

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

//go:embed automation.js
var defaultAutomationScript string

// ScriptRunner is the process boundary for JXA. Implementations used by tests
// return fixture JSON without launching osascript or contacting Things.
type ScriptRunner interface {
	Run(ctx context.Context, operation string, request any) ([]byte, error)
}

// ExecScript runs the fixed JXA program through macOS osascript. User input is
// passed as a JSON argument, never interpolated into the script source.
type ScriptLogger func(operation string, duration time.Duration, err error)

type ExecScript struct {
	Binary string
	Source string
	Logger ScriptLogger
}

func (e ExecScript) Run(ctx context.Context, operation string, request any) (output []byte, runErr error) {
	started := time.Now()
	defer func() {
		if e.Logger != nil {
			e.Logger(operation, time.Since(started), runErr)
		}
	}()
	if ctx == nil {
		return nil, &AutomationError{Kind: AutomationFailure, Operation: operation, Err: errors.New("nil context")}
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, &AutomationError{Kind: AutomationFailure, Operation: operation, Err: fmt.Errorf("encode request: %w", err)}
	}
	binary := e.Binary
	if binary == "" {
		binary = "/usr/bin/osascript"
	}
	source := e.Source
	if source == "" {
		source = defaultAutomationScript
	}

	cmd := exec.CommandContext(ctx, binary, "-l", "JavaScript", "-", operation, string(payload))
	cmd.Stdin = strings.NewReader(source)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		kind := AutomationFailure
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			kind = AutomationTimeout
		} else if errors.Is(ctx.Err(), context.Canceled) {
			kind = AutomationCanceled
		} else if looksLikePermissionFailure(stderr.String()) {
			kind = AutomationPermissionDenied
		} else if looksLikeMissingApplication(stderr.String()) {
			kind = AutomationApplicationMissing
		}
		return nil, &AutomationError{Kind: kind, Operation: operation, Err: err, Stderr: strings.TrimSpace(stderr.String())}
	}
	return stdout.Bytes(), nil
}

func looksLikeMissingApplication(stderr string) bool {
	lower := strings.ToLower(stderr)
	return strings.Contains(lower, "-2700") ||
		strings.Contains(lower, "-1728") ||
		strings.Contains(lower, "application can't be found") ||
		strings.Contains(lower, "can't get application") ||
		strings.Contains(lower, "cannot get application") ||
		strings.Contains(lower, "application isn't running")
}

func looksLikePermissionFailure(stderr string) bool {
	lower := strings.ToLower(stderr)
	return strings.Contains(lower, "-1743") ||
		strings.Contains(lower, "not authorized") ||
		strings.Contains(lower, "not allowed") ||
		strings.Contains(lower, "assistive access")
}

type AutomationFailureKind string

const (
	AutomationFailure            AutomationFailureKind = "failure"
	AutomationTimeout            AutomationFailureKind = "timeout"
	AutomationCanceled           AutomationFailureKind = "canceled"
	AutomationPermissionDenied   AutomationFailureKind = "permission_denied"
	AutomationApplicationMissing AutomationFailureKind = "application_missing"
	AutomationResponseFailure    AutomationFailureKind = "response_failure"
)

// AutomationError preserves the operation and sanitized process context while
// allowing callers to inspect the underlying process/context error.
type AutomationError struct {
	Kind      AutomationFailureKind
	Operation string
	Err       error
	Stderr    string
}

func (e *AutomationError) Error() string {
	if e == nil {
		return "automation error"
	}
	var message string
	switch e.Kind {
	case AutomationTimeout:
		message = fmt.Sprintf("Things automation timed out during %s", e.Operation)
	case AutomationCanceled:
		message = fmt.Sprintf("Things automation canceled during %s", e.Operation)
	case AutomationPermissionDenied:
		message = fmt.Sprintf("Things Automation permission was denied during %s; allow access for the invoking terminal, agent host, launcher, or executable in System Settings → Privacy & Security → Automation", e.Operation)
	case AutomationApplicationMissing:
		message = fmt.Sprintf("Things 3 is not installed or could not be opened during %s", e.Operation)
	case AutomationResponseFailure:
		message = fmt.Sprintf("Things automation returned an invalid response during %s", e.Operation)
	default:
		message = fmt.Sprintf("Things automation failed during %s", e.Operation)
	}
	if e.Err != nil {
		message += ": " + e.Err.Error()
	}
	if e.Stderr != "" {
		message += ": " + e.Stderr
	}
	return message
}

func (e *AutomationError) Unwrap() error { return e.Err }

type automationEnvelope struct {
	OK    bool                  `json:"ok"`
	Data  json.RawMessage       `json:"data"`
	Error *automationReplyError `json:"error"`
}

type automationReplyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RunJSON executes an operation and decodes the fixed script envelope into out.
// A nil out is valid for operations that only report success.
func RunJSON(ctx context.Context, runner ScriptRunner, operation string, request any, out any) error {
	if runner == nil {
		return &AutomationError{Kind: AutomationFailure, Operation: operation, Err: errors.New("automation runner is nil")}
	}
	raw, err := runner.Run(ctx, operation, request)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return responseError(operation, errors.New("empty response"))
	}
	var envelope automationEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return responseError(operation, fmt.Errorf("decode JSON: %w", err))
	}
	if !envelope.OK {
		if envelope.Error == nil || strings.TrimSpace(envelope.Error.Message) == "" {
			return responseError(operation, errors.New("response reported failure without an error"))
		}
		if envelope.Error.Code == "not_found" {
			return fmt.Errorf("%w: %s", ErrNotFound, envelope.Error.Message)
		}
		if kind, ok := automationFailureKind(envelope.Error.Code, envelope.Error.Message); ok {
			return &AutomationError{Kind: kind, Operation: operation, Err: errors.New(envelope.Error.Message)}
		}
		return &AutomationError{Kind: AutomationResponseFailure, Operation: operation, Err: errors.New(envelope.Error.Message)}
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(envelope.Data)) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return responseError(operation, errors.New("successful response has no data"))
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return responseError(operation, fmt.Errorf("decode data: %w", err))
	}
	return nil
}

func responseError(operation string, err error) error {
	return &AutomationError{Kind: AutomationResponseFailure, Operation: operation, Err: err}
}

func automationFailureKind(code, message string) (AutomationFailureKind, bool) {
	switch code {
	case "permission_denied":
		return AutomationPermissionDenied, true
	case "application_missing":
		return AutomationApplicationMissing, true
	case "timeout":
		return AutomationTimeout, true
	}
	// Older or customized JXA sources may still label the reply
	// application_error. Preserve stable categories when the raw error text is
	// an unambiguous macOS missing-application or TCC failure.
	if looksLikePermissionFailure(message) {
		return AutomationPermissionDenied, true
	}
	if looksLikeMissingApplication(message) {
		return AutomationApplicationMissing, true
	}
	return AutomationFailure, false
}

// AutomationHealth is a test-only response that proves argument transport and
// JSON envelope decoding without touching the Things application.
type AutomationHealth struct {
	Operation string         `json:"operation"`
	Request   map[string]any `json:"request"`
}
