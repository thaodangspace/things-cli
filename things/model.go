package things

import (
	"context"
	"errors"
)

type Ref struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ChecklistItem struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type Item struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Status         string          `json:"status"`
	Title          string          `json:"title"`
	Notes          string          `json:"notes,omitempty"`
	Start          string          `json:"start"`
	StartDate      *string         `json:"start_date"`
	Deadline       *string         `json:"deadline"`
	CreationDate   *string         `json:"creation_date"`
	CompletionDate *string         `json:"completion_date"`
	Area           *Ref            `json:"area"`
	Project        *Ref            `json:"project"`
	Heading        *Ref            `json:"heading"`
	Tags           []string        `json:"tags"`
	Checklist      []ChecklistItem `json:"checklist"`
	Trashed        bool            `json:"trashed"`
}

type Area struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Tag struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type ListKind string

const (
	ListNone     ListKind = ""
	ListInbox    ListKind = "inbox"
	ListToday    ListKind = "today"
	ListUpcoming ListKind = "upcoming"
	ListAnytime  ListKind = "anytime"
	ListSomeday  ListKind = "someday"
	ListLogbook  ListKind = "logbook"
	ListTrash    ListKind = "trash"
)

type Filter struct {
	Status  string
	Type    string
	Tag     string
	Area    string
	Project string
	Limit   int
	List    ListKind
}

var ErrNotFound = errors.New("item not found")

type ActionResult struct {
	Action string `json:"action"`
	ID     string `json:"id,omitempty"`
}

// Service is the storage-neutral boundary used by CLI commands. Implementations
// may use Things automation, while tests can provide deterministic fakes.
type Service interface {
	List(ctx context.Context, list ListKind, limit int) ([]Item, error)
	Query(ctx context.Context, filter Filter) ([]Item, error)
	Get(ctx context.Context, id string) (Item, error)
	ListProjects(ctx context.Context, area string, limit int) ([]Item, error)
	ListAreas(ctx context.Context) ([]Area, error)
	ListTags(ctx context.Context) ([]Tag, error)
	Add(ctx context.Context, request AddRequest) (ActionResult, error)
	AddProject(ctx context.Context, request AddProjectRequest) (ActionResult, error)
	Update(ctx context.Context, request UpdateRequest) (ActionResult, error)
	Complete(ctx context.Context, id string) (ActionResult, error)
	Cancel(ctx context.Context, id string) (ActionResult, error)
	Show(ctx context.Context, target string) (ActionResult, error)
	Search(ctx context.Context, query string) (ActionResult, error)
}

type AddRequest struct {
	Title     string   `json:"title"`
	Notes     string   `json:"notes"`
	When      string   `json:"when"`
	Deadline  string   `json:"deadline"`
	Tags      []string `json:"tags"`
	List      string   `json:"list"`
	ListID    string   `json:"list_id"`
	Completed bool     `json:"completed"`
	Canceled  bool     `json:"canceled"`
	Reveal    bool     `json:"reveal"`
}

type AddProjectRequest struct {
	Title    string   `json:"title"`
	Notes    string   `json:"notes"`
	When     string   `json:"when"`
	Deadline string   `json:"deadline"`
	Tags     []string `json:"tags"`
	Area     string   `json:"area"`
	AreaID   string   `json:"area_id"`
	ToDos    []string `json:"to_dos"`
	Reveal   bool     `json:"reveal"`
}

type UpdateRequest struct {
	ID           string    `json:"id"`
	Title        *string   `json:"title,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	PrependNotes *string   `json:"prepend_notes,omitempty"`
	AppendNotes  *string   `json:"append_notes,omitempty"`
	When         *string   `json:"when,omitempty"`
	Deadline     *string   `json:"deadline,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
	AddTags      *[]string `json:"add_tags,omitempty"`
	List         *string   `json:"list,omitempty"`
	ListID       *string   `json:"list_id,omitempty"`
	Completed    *bool     `json:"completed,omitempty"`
	Canceled     *bool     `json:"canceled,omitempty"`
	Project      bool      `json:"project"`
}
