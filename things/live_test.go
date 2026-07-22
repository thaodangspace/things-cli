package things

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveReadOnlyAutomation is opt-in because it requires Things 3 and
// Automation permission and may expose a user's item counts to test output.
func TestLiveReadOnlyAutomation(t *testing.T) {
	if os.Getenv("THINGS_LIVE_TEST") != "1" {
		t.Skip("set THINGS_LIVE_TEST=1 to run read-only Things automation smoke tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := NewAutomationClient(nil)
	items, err := client.List(ctx, ListToday, 1)
	if err != nil {
		t.Fatal(err)
	}
	areas, err := client.ListAreas(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tags, err := client.ListTags(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if items == nil || areas == nil || tags == nil {
		t.Fatal("automation returned nil collection")
	}
	for _, item := range items {
		if item.ID == "" || item.Type == "" || item.Status == "" {
			t.Fatalf("invalid item shape: %+v", item)
		}
	}
	for _, area := range areas {
		if area.ID == "" {
			t.Fatalf("invalid area shape: %+v", area)
		}
	}
	for _, tag := range tags {
		if tag.ID == "" {
			t.Fatalf("invalid tag shape: %+v", tag)
		}
	}
}
