package things

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AutomationClient implements the Things service over the public scripting
// interface. Its runner is injectable so unit tests never launch osascript.
type AutomationClient struct {
	runner ScriptRunner
}

func NewAutomationClient(runner ScriptRunner) *AutomationClient {
	if runner == nil {
		runner = ExecScript{}
	}
	return &AutomationClient{runner: runner}
}

type listRequest struct {
	List  ListKind `json:"list"`
	Limit int      `json:"limit"`
}

type queryRequest struct {
	Status         string   `json:"status,omitempty"`
	Type           string   `json:"type,omitempty"`
	Tag            string   `json:"tag,omitempty"`
	Area           string   `json:"area,omitempty"`
	Project        string   `json:"project,omitempty"`
	Text           string   `json:"text,omitempty"`
	CreatedAfter   string   `json:"created_after,omitempty"`
	CreatedBefore  string   `json:"created_before,omitempty"`
	ModifiedAfter  string   `json:"modified_after,omitempty"`
	ModifiedBefore string   `json:"modified_before,omitempty"`
	DeadlineAfter  string   `json:"deadline_after,omitempty"`
	DeadlineBefore string   `json:"deadline_before,omitempty"`
	StartAfter     string   `json:"start_after,omitempty"`
	StartBefore    string   `json:"start_before,omitempty"`
	Sort           string   `json:"sort,omitempty"`
	Reverse        bool     `json:"reverse,omitempty"`
	All            bool     `json:"all,omitempty"`
	Limit          int      `json:"limit"`
	List           ListKind `json:"list,omitempty"`
}

type projectsRequest struct {
	Area  string `json:"area,omitempty"`
	Limit int    `json:"limit"`
}

type idRequest struct {
	ID string `json:"id"`
}

func (c *AutomationClient) run(ctx context.Context, operation string, request any, out any) error {
	return RunJSON(ctx, c.runner, operation, request, out)
}

func (c *AutomationClient) List(ctx context.Context, list ListKind, limit int) ([]Item, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []Item
	if err := c.run(ctx, "list", listRequest{List: list, Limit: limit}, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []Item{}
	}
	return items, nil
}

func (c *AutomationClient) Query(ctx context.Context, filter Filter) ([]Item, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	if filter.All && filter.Limit > 0 {
		return nil, fmt.Errorf("--all cannot be combined with --limit")
	}
	if !filter.All && filter.Limit <= 0 {
		filter.Limit = 50
	}
	status := filter.Status
	if status != "" {
		status, _ = StatusValue(status)
	}
	itemType := filter.Type
	if itemType != "" {
		itemType, _ = TypeValue(itemType)
	}
	request := queryRequest{
		Status: status, Type: itemType, Tag: filter.Tag,
		Area: filter.Area, Project: filter.Project, Text: filter.Text,
		CreatedAfter: filter.CreatedAfter, CreatedBefore: filter.CreatedBefore,
		ModifiedAfter: filter.ModifiedAfter, ModifiedBefore: filter.ModifiedBefore,
		DeadlineAfter: filter.DeadlineAfter, DeadlineBefore: filter.DeadlineBefore,
		StartAfter: filter.StartAfter, StartBefore: filter.StartBefore,
		Sort: filter.Sort, Reverse: filter.Reverse, All: filter.All,
		Limit: filter.Limit, List: filter.List,
	}
	var items []Item
	if err := c.run(ctx, "query", request, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []Item{}
	}
	return items, nil
}

func (c *AutomationClient) Get(ctx context.Context, id string) (Item, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Item{}, fmt.Errorf("item id is required")
	}
	var item Item
	if err := c.run(ctx, "get", idRequest{ID: id}, &item); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (c *AutomationClient) ListProjects(ctx context.Context, area string, limit int) ([]Item, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []Item
	if err := c.run(ctx, "list-projects", projectsRequest{Area: area, Limit: limit}, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []Item{}
	}
	return items, nil
}

func (c *AutomationClient) ListAreas(ctx context.Context) ([]Area, error) {
	var areas []Area
	if err := c.run(ctx, "list-areas", struct{}{}, &areas); err != nil {
		return nil, err
	}
	if areas == nil {
		areas = []Area{}
	}
	return areas, nil
}

func (c *AutomationClient) ListTags(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	if err := c.run(ctx, "list-tags", struct{}{}, &tags); err != nil {
		return nil, err
	}
	if tags == nil {
		tags = []Tag{}
	}
	return tags, nil
}

func validateFilter(filter Filter) error {
	if filter.Status != "" {
		if _, ok := StatusValue(filter.Status); !ok {
			return fmt.Errorf("unknown status %q", filter.Status)
		}
	}
	if filter.Type != "" {
		if _, ok := TypeValue(filter.Type); !ok {
			return fmt.Errorf("unknown type %q", filter.Type)
		}
	}
	if strings.TrimSpace(filter.Text) == "" && filter.Text != "" {
		return fmt.Errorf("text filter may not be empty")
	}
	if filter.Sort != "" {
		switch filter.Sort {
		case "native", "title", "created", "modified", "deadline", "start":
		default:
			return fmt.Errorf("unknown sort %q", filter.Sort)
		}
	}
	for _, field := range []struct {
		name     string
		value    string
		dateOnly bool
	}{
		{"created-after", filter.CreatedAfter, false},
		{"created-before", filter.CreatedBefore, false},
		{"modified-after", filter.ModifiedAfter, false},
		{"modified-before", filter.ModifiedBefore, false},
		{"deadline-after", filter.DeadlineAfter, true},
		{"deadline-before", filter.DeadlineBefore, true},
		{"start-after", filter.StartAfter, true},
		{"start-before", filter.StartBefore, true},
	} {
		if field.value == "" {
			continue
		}
		if field.dateOnly {
			if _, err := time.Parse("2006-01-02", field.value); err != nil {
				return fmt.Errorf("invalid %s %q; use yyyy-mm-dd", field.name, field.value)
			}
		} else if _, err := time.Parse(time.RFC3339, field.value); err != nil {
			if _, dateErr := time.Parse("2006-01-02", field.value); dateErr != nil {
				return fmt.Errorf("invalid %s %q; use RFC3339", field.name, field.value)
			}
		}
	}
	return nil
}

// Write and navigation operations use the same fixed automation boundary as
// reads. They are intentionally synchronous: a successful response means the
// Things script handled the operation.
func (c *AutomationClient) Add(ctx context.Context, request AddRequest) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "add", request, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) AddProject(ctx context.Context, request AddProjectRequest) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "add-project", request, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Update(ctx context.Context, request UpdateRequest) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "update", request, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Move(ctx context.Context, request MoveRequest) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "move", request, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Detach(ctx context.Context, request DetachRequest) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "detach", request, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Delete(ctx context.Context, id string) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "delete", idRequest{ID: id}, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) EmptyTrash(ctx context.Context) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "empty-trash", struct{}{}, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Complete(ctx context.Context, id string) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "complete", idRequest{ID: id}, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Cancel(ctx context.Context, id string) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "cancel", idRequest{ID: id}, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Show(ctx context.Context, target string) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "show", struct {
		Target string `json:"target"`
	}{Target: target}, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

func (c *AutomationClient) Search(ctx context.Context, query string) (ActionResult, error) {
	var result ActionResult
	if err := c.run(ctx, "search", struct {
		Query string `json:"query"`
	}{Query: query}, &result); err != nil {
		return ActionResult{}, err
	}
	return result, nil
}

var _ Service = (*AutomationClient)(nil)
