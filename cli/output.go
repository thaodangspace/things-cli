package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/thaodangspace/things-cli/things"
)

type successEnvelope struct {
	OK   bool `json:"ok"`
	Data any  `json:"data"`
}

type errorEnvelope struct {
	OK    bool        `json:"ok"`
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Message string `json:"message"`
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeSuccess(w io.Writer, data any, human bool, humanText string) error {
	if human {
		_, err := fmt.Fprintln(w, humanText)
		return err
	}
	return writeJSON(w, successEnvelope{OK: true, Data: data})
}

func writeError(w io.Writer, err error, human bool) {
	if human {
		fmt.Fprintln(w, err.Error())
		return
	}
	_ = writeJSON(w, errorEnvelope{OK: false, Error: errorDetail{Message: err.Error()}})
}

func summarizeItem(it things.Item) string {
	title := it.Title
	if title == "" {
		title = "(untitled)"
	}
	return fmt.Sprintf("%s %s [%s %s]", orUnknown(it.ID), title, it.Type, it.Status)
}

func summarizeArea(a things.Area) string {
	return fmt.Sprintf("%s %s", orUnknown(a.ID), defaultString(a.Title, "(untitled area)"))
}
func summarizeTag(t things.Tag) string {
	return fmt.Sprintf("%s %s", orUnknown(t.ID), defaultString(t.Title, "(untitled tag)"))
}

func summarizeAction(a things.ActionResult) string {
	msg := a.Action
	if a.ID != "" {
		msg += " " + a.ID
	}
	return strings.TrimSpace(msg)
}

func humanItemList(items []things.Item, emptyMsg string) string {
	if len(items) == 0 {
		return emptyMsg
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, summarizeItem(it))
	}
	return strings.Join(lines, "\n")
}

func humanAreaList(items []things.Area, emptyMsg string) string {
	if len(items) == 0 {
		return emptyMsg
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, summarizeArea(it))
	}
	return strings.Join(lines, "\n")
}

func humanTagList(items []things.Tag, emptyMsg string) string {
	if len(items) == 0 {
		return emptyMsg
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, summarizeTag(it))
	}
	return strings.Join(lines, "\n")
}

func defaultString(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func orUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}
