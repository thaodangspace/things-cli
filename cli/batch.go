package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thaodangspace/things-cli/things"
)

const (
	// These limits are deliberately conservative defaults for generated plans.
	// They bound memory used by the line scanner and prevent an accidental
	// unbounded write stream from running unattended.
	batchMaxLineBytes  = 1 << 20 // 1 MiB, excluding the newline
	batchMaxOperations = 1000
)

type batchInput struct {
	ClientID  json.RawMessage `json:"client_id"`
	Operation string          `json:"operation"`
	Request   json.RawMessage `json:"request"`
}

type batchEntry struct {
	clientID  *string
	operation string
	request   json.RawMessage
}

type batchOperation struct {
	decode  func(json.RawMessage) (any, error)
	execute func(context.Context, things.Service, any) (things.ActionResult, error)
}

type batchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type batchResult struct {
	Index    int         `json:"index"`
	ClientID *string     `json:"client_id,omitempty"`
	OK       bool        `json:"ok"`
	Data     any         `json:"data,omitempty"`
	Error    *batchError `json:"error,omitempty"`
}

type normalizedBatchRequest struct {
	Operation string `json:"operation"`
	Request   any    `json:"request"`
}

func newBatchCommand() *cobra.Command {
	var inputPath string
	var continueOnError, validateOnly bool
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Execute sequential write operations from an NDJSON stream",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 0 {
				return usageErrorf("batch does not accept positional arguments")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.human {
				return usageErrorf("--human is not supported with batch; batch output is NDJSON")
			}
			var input io.Reader = cmd.InOrStdin()
			var file *os.File
			if inputPath != "-" {
				var err error
				file, err = os.Open(inputPath)
				if err != nil {
					return usageErrorf("open batch input %q: %v", inputPath, err)
				}
				defer file.Close()
				input = file
			}
			return runBatch(cmd, input, continueOnError, validateOnly)
		},
	}
	cmd.Flags().StringVar(&inputPath, "input", "-", "read NDJSON from FILE, or - for stdin")
	cmd.Flags().BoolVar(&continueOnError, "continue-on-error", false, "continue after an operation or validation error")
	cmd.Flags().BoolVar(&validateOnly, "validate-only", false, "validate the complete stream without invoking Things")
	return cmd
}

func runBatch(cmd *cobra.Command, input io.Reader, continueOnError, validateOnly bool) error {
	reader := bufio.NewReaderSize(input, 64*1024)
	registry := batchOperationRegistry()
	lineNumber := 0
	operationIndex := 0
	invalidInput := false
	var firstRuntime error

	emit := func(result batchResult) error {
		// Unlike normal command JSON, batch output must be exactly one compact
		// JSON object per line.
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}
	emitError := func(index int, clientID *string, code, message string) error {
		return emit(batchResult{Index: index, ClientID: clientID, OK: false, Error: &batchError{Code: code, Message: message}})
	}

	for {
		line, ok, tooLarge, readErr := readBatchLine(reader)
		if readErr != nil {
			operationIndex++
			if firstRuntime == nil {
				firstRuntime = readErr
			}
			if emitErr := emitError(operationIndex, nil, "input_error", readErr.Error()); emitErr != nil {
				return emitErr
			}
			break
		}
		if !ok {
			break
		}
		lineNumber++
		if tooLarge {
			operationIndex++
			invalidInput = true
			if err := emitError(operationIndex, nil, "line_too_large", fmt.Sprintf("line %d exceeds maximum size of %d bytes", lineNumber, batchMaxLineBytes)); err != nil {
				return err
			}
			if !validateOnly && !continueOnError {
				break
			}
			continue
		}
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if operationIndex >= batchMaxOperations {
			operationIndex++
			invalidInput = true
			if err := emitError(operationIndex, nil, "too_many_operations", fmt.Sprintf("line %d exceeds maximum operation count of %d", lineNumber, batchMaxOperations)); err != nil {
				return err
			}
			break
		}
		operationIndex++
		entry, err := parseBatchEntry(line)
		if err != nil {
			invalidInput = true
			if emitErr := emitError(operationIndex, entry.clientID, batchErrorCode(err), fmt.Sprintf("line %d: %v", lineNumber, err)); emitErr != nil {
				return emitErr
			}
			if !validateOnly && !continueOnError {
				break
			}
			continue
		}

		if strings.TrimSpace(entry.operation) == "" {
			invalidInput = true
			if emitErr := emitError(operationIndex, entry.clientID, "invalid_request", fmt.Sprintf("line %d: operation is required", lineNumber)); emitErr != nil {
				return emitErr
			}
			if !validateOnly && !continueOnError {
				break
			}
			continue
		}
		spec, ok := registry[entry.operation]
		if !ok {
			invalidInput = true
			if emitErr := emitError(operationIndex, entry.clientID, "unknown_operation", fmt.Sprintf("line %d: unknown operation %q", lineNumber, entry.operation)); emitErr != nil {
				return emitErr
			}
			if !validateOnly && !continueOnError {
				break
			}
			continue
		}
		request, err := spec.decode(entry.request)
		if err != nil {
			invalidInput = true
			if emitErr := emitError(operationIndex, entry.clientID, "invalid_request", fmt.Sprintf("line %d: %v", lineNumber, err)); emitErr != nil {
				return emitErr
			}
			if !validateOnly && !continueOnError {
				break
			}
			continue
		}

		if validateOnly {
			if err := emit(batchResult{
				Index: operationIndex, ClientID: entry.clientID, OK: true,
				Data: normalizedBatchRequest{Operation: entry.operation, Request: request},
			}); err != nil {
				return err
			}
			continue
		}

		ctx, cancel := withTimeout(cmd)
		result, err := spec.execute(ctx, currentThingsService(), request)
		cancel()
		if err != nil {
			if firstRuntime == nil {
				firstRuntime = err
			}
			code := batchErrorCode(err)
			if emitErr := emitError(operationIndex, entry.clientID, code, err.Error()); emitErr != nil {
				return emitErr
			}
			if !continueOnError {
				break
			}
			continue
		}
		if err := emit(batchResult{Index: operationIndex, ClientID: entry.clientID, OK: true, Data: result}); err != nil {
			return err
		}
	}

	if invalidInput {
		return usageErrorf("batch contains invalid input")
	}
	if firstRuntime != nil {
		return firstRuntime
	}
	return nil
}

// readBatchLine reads one physical line while retaining only the documented
// maximum. If the line is too large it drains the remaining fragments before
// returning, allowing callers to continue with the next line.
func readBatchLine(reader *bufio.Reader) (line []byte, ok, tooLarge bool, err error) {
	var haveData bool
	for {
		fragment, prefix, readErr := reader.ReadLine()
		if readErr != nil && readErr != io.EOF {
			return nil, false, false, readErr
		}
		if readErr == nil {
			haveData = true
		}
		if len(fragment) > 0 {
			haveData = true
		}
		if !tooLarge {
			if len(line)+len(fragment) > batchMaxLineBytes {
				tooLarge = true
				line = nil
			} else {
				line = append(line, fragment...)
			}
		}
		if readErr == io.EOF || !prefix {
			if readErr == io.EOF && !haveData {
				return nil, false, false, nil
			}
			return line, true, tooLarge, nil
		}
	}
}

// parseBatchEntry validates the envelope and returns the raw request. Keeping
// the request raw until the operation registry is selected gives each
// operation one strict, typed schema. The client ID is parsed before later
// envelope checks so it can still be returned with a validation error.
func parseBatchEntry(line []byte) (batchEntry, error) {
	var wire batchInput
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return batchEntry{}, &batchInputError{code: "invalid_json", msg: fmt.Sprintf("malformed JSON: %v", err)}
	}
	clientID, err := parseBatchClientID(wire.ClientID)
	entry := batchEntry{clientID: clientID, operation: wire.Operation, request: wire.Request}
	if err != nil {
		return entry, &batchInputError{code: "invalid_request", msg: err.Error()}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return entry, &batchInputError{code: "invalid_json", msg: "malformed JSON: multiple values on one line"}
		}
		return entry, &batchInputError{code: "invalid_json", msg: fmt.Sprintf("malformed JSON: %v", err)}
	}
	if len(wire.Request) == 0 || bytes.Equal(bytes.TrimSpace(wire.Request), []byte("null")) {
		return entry, &batchInputError{code: "invalid_request", msg: "request is required and must be an object"}
	}
	return entry, nil
}

func parseBatchClientID(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, errors.New("client_id must be a string")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, errors.New("client_id must be a string")
	}
	return &value, nil
}

func decodeBatchRequest[T any](raw json.RawMessage) (T, error) {
	var request T
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return request, errors.New("request is required and must be an object")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("invalid request: %v", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return request, errors.New("invalid request: multiple values")
		}
		return request, fmt.Errorf("invalid request: %v", err)
	}
	return request, nil
}

func batchOperationRegistry() map[string]batchOperation {
	return map[string]batchOperation{
		"add": {
			decode: func(raw json.RawMessage) (any, error) {
				request, err := decodeBatchRequest[things.AddRequest](raw)
				if err != nil {
					return nil, err
				}
				normalized, err := validateAddRequest(request)
				return normalized, err
			},
			execute: func(ctx context.Context, service things.Service, value any) (things.ActionResult, error) {
				return service.Add(ctx, value.(things.AddRequest))
			},
		},
		"add-project": {
			decode: func(raw json.RawMessage) (any, error) {
				request, err := decodeBatchRequest[things.AddProjectRequest](raw)
				if err != nil {
					return nil, err
				}
				normalized, err := validateAddProjectRequest(request)
				return normalized, err
			},
			execute: func(ctx context.Context, service things.Service, value any) (things.ActionResult, error) {
				return service.AddProject(ctx, value.(things.AddProjectRequest))
			},
		},
		"update": {
			decode: func(raw json.RawMessage) (any, error) {
				request, err := decodeBatchRequest[things.UpdateRequest](raw)
				if err != nil {
					return nil, err
				}
				normalized, err := validateUpdateRequest(request)
				return normalized, err
			},
			execute: func(ctx context.Context, service things.Service, value any) (things.ActionResult, error) {
				return service.Update(ctx, value.(things.UpdateRequest))
			},
		},
		"complete": batchIDOperation(true),
		"cancel":   batchIDOperation(false),
	}
}

type batchIDRequest struct {
	ID string `json:"id"`
}

func batchIDOperation(complete bool) batchOperation {
	return batchOperation{
		decode: func(raw json.RawMessage) (any, error) {
			request, err := decodeBatchRequest[batchIDRequest](raw)
			if err != nil {
				return nil, err
			}
			request.ID = strings.TrimSpace(request.ID)
			if request.ID == "" {
				return nil, usageErrorf("id is required")
			}
			return request, nil
		},
		execute: func(ctx context.Context, service things.Service, value any) (things.ActionResult, error) {
			request := value.(batchIDRequest)
			if complete {
				return service.Complete(ctx, request.ID)
			}
			return service.Cancel(ctx, request.ID)
		},
	}
}

func batchErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var input *batchInputError
	if errors.As(err, &input) {
		return input.code
	}
	var usage *usageError
	if errors.As(err, &usage) {
		return "invalid_request"
	}
	if errors.Is(err, things.ErrNotFound) {
		return "not_found"
	}
	var automation *things.AutomationError
	if errors.As(err, &automation) {
		switch automation.Kind {
		case things.AutomationTimeout:
			return "timeout"
		case things.AutomationPermissionDenied:
			return "permission_denied"
		case things.AutomationApplicationMissing:
			return "application_missing"
		case things.AutomationResponseFailure:
			return "application_error"
		default:
			return "automation_error"
		}
	}
	return "runtime_error"
}

type batchInputError struct {
	code string
	msg  string
}

func (e *batchInputError) Error() string { return e.msg }
