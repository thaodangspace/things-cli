package things

import (
	"context"
	"fmt"
	"strings"
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
	Status  string   `json:"status,omitempty"`
	Type    string   `json:"type,omitempty"`
	Tag     string   `json:"tag,omitempty"`
	Area    string   `json:"area,omitempty"`
	Project string   `json:"project,omitempty"`
	Limit   int      `json:"limit"`
	List    ListKind `json:"list,omitempty"`
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
	if filter.Limit <= 0 {
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
		Area: filter.Area, Project: filter.Project, Limit: filter.Limit, List: filter.List,
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
