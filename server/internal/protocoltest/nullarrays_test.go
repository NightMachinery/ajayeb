package protocoltest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// recordedMessagesDir is where internal/wire's own recorder (record_test.go,
// go test ./internal/wire -run TestRecordMessages -update) writes real
// server frames. ajayeb-gen fixtures turns this directory into
// web/src/lib/wire/fixtures.gen.ts.
const recordedMessagesDir = "../wire/testdata/messages"

// TestRecordedMessagesHaveNoJSONNulls walks every recorded message and
// fails on any JSON null, anywhere in the document. Package wire's own
// rule (doc.go) is that a slice field is never nil, so it always marshals
// as [] rather than null, and every pointer field carries omitempty, so an
// absent optional value is a missing key, never a present key holding
// null. A null surviving in a recorded message is exactly the mistake that
// rule exists to prevent, and it would reach the generated TypeScript
// fixture (ajayeb-gen fixtures) and fail svelte-check there instead of
// here, for a less obvious reason.
func TestRecordedMessagesHaveNoJSONNulls(t *testing.T) {
	entries, err := os.ReadDir(recordedMessagesDir)
	if err != nil {
		t.Fatalf("ReadDir(%s): %v", recordedMessagesDir, err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(recordedMessagesDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatalf("Unmarshal(%s): %v", path, err)
		}
		checked++
		findJSONNulls(t, e.Name(), "$", v)
	}
	if checked == 0 {
		t.Fatalf("no recorded messages found under %s", recordedMessagesDir)
	}
}

// findJSONNulls reports every JSON null in v, labeled with its path (a
// jq-like "$.room.seats[0].bot" trail) and the file it came from.
func findJSONNulls(t *testing.T, file, path string, v any) {
	t.Helper()
	switch val := v.(type) {
	case nil:
		t.Errorf("%s: %s is JSON null, want it omitted (a nil pointer with omitempty) or an empty array (a non-nil, empty slice)", file, path)
	case map[string]any:
		for k, child := range val {
			findJSONNulls(t, file, path+"."+k, child)
		}
	case []any:
		for i, child := range val {
			findJSONNulls(t, file, fmt.Sprintf("%s[%d]", path, i), child)
		}
	}
}
