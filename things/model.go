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
	ID               string          `json:"id"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	Title            string          `json:"title"`
	Notes            string          `json:"notes,omitempty"`
	Start            string          `json:"start"`
	StartDate        *string         `json:"start_date"`
	Deadline         *string         `json:"deadline"`
	CreationDate     *string         `json:"creation_date"`
	ModificationDate *string         `json:"modification_date"`
	CompletionDate   *string         `json:"completion_date"`
	Area             *Ref            `json:"area"`
	Project          *Ref            `json:"project"`
	Heading          *Ref            `json:"heading"`
	Tags             []string        `json:"tags"`
	Checklist        []ChecklistItem `json:"checklist"`
	Trashed          bool            `json:"trashed"`
}

type Area struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Tag struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Parent *Ref   `json:"parent,omitempty"`
}

type ResourceTarget struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type AddAreaRequest struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

type RenameAreaRequest struct {
	Target ResourceTarget `json:"target"`
	Title  string         `json:"title"`
}

type DeleteAreaRequest struct {
	Target ResourceTarget `json:"target"`
}

type AddTagRequest struct {
	Title  string         `json:"title"`
	Parent ResourceTarget `json:"parent,omitempty"`
}

type RenameTagRequest struct {
	Target ResourceTarget `json:"target"`
	Title  string         `json:"title"`
}

type SetTagParentRequest struct {
	Target ResourceTarget `json:"target"`
	Parent ResourceTarget `json:"parent,omitempty"`
	Root   bool           `json:"root,omitempty"`
}

type DeleteTagRequest struct {
	Target ResourceTarget `json:"target"`
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
	Status         string
	Type           string
	Tag            string
	Area           string
	Project        string
	Text           string
	CreatedAfter   string
	CreatedBefore  string
	ModifiedAfter  string
	ModifiedBefore string
	DeadlineAfter  string
	DeadlineBefore string
	StartAfter     string
	StartBefore    string
	Sort           string
	Reverse        bool
	All            bool
	Limit          int
	List           ListKind
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
	Move(ctx context.Context, request MoveRequest) (ActionResult, error)
	Detach(ctx context.Context, request DetachRequest) (ActionResult, error)
	Delete(ctx context.Context, id string) (ActionResult, error)
	EmptyTrash(ctx context.Context) (ActionResult, error)
	Complete(ctx context.Context, id string) (ActionResult, error)
	Cancel(ctx context.Context, id string) (ActionResult, error)
	Show(ctx context.Context, target string) (ActionResult, error)
	Search(ctx context.Context, query string) (ActionResult, error)
}

// AreaTagService contains the resource-management operations added after the
// original Service contract. Keeping it separate preserves compatibility with
// integrations that only implement the original read/write surface.
type AreaTagService interface {
	Service
	ListTagsTree(ctx context.Context) ([]Tag, error)
	AddArea(ctx context.Context, request AddAreaRequest) (ActionResult, error)
	RenameArea(ctx context.Context, request RenameAreaRequest) (ActionResult, error)
	DeleteArea(ctx context.Context, request DeleteAreaRequest) (ActionResult, error)
	AddTag(ctx context.Context, request AddTagRequest) (ActionResult, error)
	RenameTag(ctx context.Context, request RenameTagRequest) (ActionResult, error)
	SetTagParent(ctx context.Context, request SetTagParentRequest) (ActionResult, error)
	DeleteTag(ctx context.Context, request DeleteTagRequest) (ActionResult, error)
}

type AddRequest struct {
	Title     string   `json:"title"`
	Notes     string   `json:"notes"`
	When      string   `json:"when"`
	Deadline  string   `json:"deadline"`
	Tags      []string `json:"tags"`
	List      string   `json:"list"`
	ListID    string   `json:"list_id"`
	Project   string   `json:"project"`
	ProjectID string   `json:"project_id"`
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

// MoveRequest relocates an item to a built-in list, project, or area. Exactly
// one destination group (list, project, or area) must be supplied.
type MoveRequest struct {
	ID        string `json:"id"`
	List      string `json:"list,omitempty"`
	ListID    string `json:"list_id,omitempty"`
	Project   string `json:"project,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Area      string `json:"area,omitempty"`
	AreaID    string `json:"area_id,omitempty"`
}

// DetachRequest removes an item's project and/or area relationships. The CLI
// expands --all into both Project and Area true.
type DetachRequest struct {
	ID      string `json:"id"`
	Project bool   `json:"project"`
	Area    bool   `json:"area"`
	All     bool   `json:"all,omitempty"`
}
