package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/thaodangspace/things-cli/things"
)

func TestWriteSuccessJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := writeSuccess(&buf, map[string]string{"id": "1"}, false, "ignored"); err != nil {
		t.Fatal(err)
	}
	var env struct {
		OK   bool              `json:"ok"`
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !env.OK || env.Data["id"] != "1" {
		t.Fatalf("unexpected envelope: %s", buf.String())
	}
}

func TestWriteSuccessHuman(t *testing.T) {
	var buf bytes.Buffer
	if err := writeSuccess(&buf, nil, true, "hello"); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "hello" || strings.Contains(buf.String(), "{") {
		t.Fatalf("out = %q", buf.String())
	}
}

func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	writeError(&buf, errors.New("boom"), false)
	var env struct {
		OK    bool `json:"ok"`
		Error struct{ Message string }
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.OK || env.Error.Message != "boom" {
		t.Fatalf("env = %+v", env)
	}
}

func TestSummaries(t *testing.T) {
	item := things.Item{ID: "1", Title: "Ship", Type: "to-do", Status: "open"}
	if got := summarizeItem(item); got != "1 Ship [to-do open]" {
		t.Fatalf("item = %q", got)
	}
	if got := humanItemList(nil, "none"); got != "none" {
		t.Fatalf("empty = %q", got)
	}
	if got := humanItemList([]things.Item{item}, "none"); got != "1 Ship [to-do open]" {
		t.Fatalf("list = %q", got)
	}
}
