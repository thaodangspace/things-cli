package cli

import (
	"strings"

	"github.com/thaodangspace/things-cli/things"
)

// validateAddRequest is shared by the flag and batch interfaces. It also
// applies the same title normalization as the flag interface.
func validateAddRequest(request things.AddRequest) (things.AddRequest, error) {
	title, err := requireFlag("title", request.Title)
	if err != nil {
		return things.AddRequest{}, err
	}
	request.Title = title
	if err := validateWhen(request.When); err != nil {
		return things.AddRequest{}, err
	}
	if err := validateDeadline(request.Deadline); err != nil {
		return things.AddRequest{}, err
	}
	if request.Completed && request.Canceled {
		return things.AddRequest{}, usageErrorf("--completed and --canceled cannot both be set")
	}
	if (request.Project != "" || request.ProjectID != "") && (request.List != "" || request.ListID != "") {
		return things.AddRequest{}, usageErrorf("--project/--project-id cannot be combined with --list/--list-id")
	}
	if err := validateCaps(request.Title, request.Notes, nil); err != nil {
		return things.AddRequest{}, err
	}
	return request, nil
}

func validateAddProjectRequest(request things.AddProjectRequest) (things.AddProjectRequest, error) {
	title, err := requireFlag("title", request.Title)
	if err != nil {
		return things.AddProjectRequest{}, err
	}
	request.Title = title
	if err := validateWhen(request.When); err != nil {
		return things.AddProjectRequest{}, err
	}
	if err := validateDeadline(request.Deadline); err != nil {
		return things.AddProjectRequest{}, err
	}
	if err := validateCaps(request.Title, request.Notes, request.ToDos); err != nil {
		return things.AddProjectRequest{}, err
	}
	return request, nil
}

func validateUpdateRequest(request things.UpdateRequest) (things.UpdateRequest, error) {
	request.ID = strings.TrimSpace(request.ID)
	if request.ID == "" {
		return things.UpdateRequest{}, usageErrorf("update requires a non-empty id")
	}
	if request.Title != nil && *request.Title == "" {
		return things.UpdateRequest{}, usageErrorf("--title may not be empty")
	}
	if err := validateWhenValue(request.When); err != nil {
		return things.UpdateRequest{}, err
	}
	if err := validateDeadlineValue(request.Deadline); err != nil {
		return things.UpdateRequest{}, err
	}
	if request.Completed != nil && request.Canceled != nil {
		return things.UpdateRequest{}, usageErrorf("--completed and --canceled cannot both be set")
	}
	var title, notes string
	if request.Title != nil {
		title = *request.Title
	}
	if request.Notes != nil {
		notes = *request.Notes
	}
	if err := validateCaps(title, notes, nil); err != nil {
		return things.UpdateRequest{}, err
	}
	return request, nil
}

// The flag validators accept empty values as "not supplied". Batch update
// requests use pointers to preserve explicit empty strings, so these helpers
// keep that distinction while reusing the date rules.
func validateWhenValue(value *string) error {
	if value == nil {
		return nil
	}
	return validateWhen(*value)
}

func validateDeadlineValue(value *string) error {
	if value == nil {
		return nil
	}
	return validateDeadline(*value)
}
