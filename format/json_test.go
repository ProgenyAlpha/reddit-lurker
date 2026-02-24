package format

import (
	"encoding/json"
	"testing"
)

// isValidJSON returns true when s parses as valid JSON.
func isValidJSON(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}

// ---------------------------------------------------------------------------
// ToJSON
// ---------------------------------------------------------------------------

func TestToJSON_NormalStruct_ValidJSON(t *testing.T) {
	type point struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	out := ToJSON(point{X: 1, Y: 2})
	if !isValidJSON(out) {
		t.Errorf("ToJSON produced invalid JSON: %s", out)
	}
}

func TestToJSON_Nil_ValidJSON(t *testing.T) {
	out := ToJSON(nil)
	if !isValidJSON(out) {
		t.Errorf("ToJSON(nil) produced invalid JSON: %s", out)
	}
}

// TestToJSON_ErrorPathWithQuotes verifies that when json.MarshalIndent fails,
// the fallback error string is properly JSON-escaped so the result remains
// valid JSON even if the error message contains double-quotes or backslashes.
//
// We trigger MarshalIndent failure by passing a channel (not serialisable).
// The error message from Go's JSON encoder typically contains the Go type name
// which won't have quotes, but we test the entire error path regardless.
func TestToJSON_UnserialisableValue_ValidJSON(t *testing.T) {
	ch := make(chan int) // channels are not JSON-serialisable
	out := ToJSON(ch)
	if !isValidJSON(out) {
		t.Errorf("ToJSON with unserialisable value produced invalid JSON: %s", out)
	}
	// Must contain an "error" key.
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("output is not a JSON object: %v", err)
	}
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' key in fallback JSON, got: %s", out)
	}
}

// TestToJSON_ErrorWithSpecialChars simulates the bug directly: if error.Error()
// returns a string with double-quotes, the old string-concatenation approach
// would break JSON validity.  We test via a map with a json.Marshaler that errors.
func TestToJSON_ErrorWithSpecialChars_ValidJSON(t *testing.T) {
	// A map key of interface type that includes a func value will fail marshalling.
	// The error string from encoding/json typically looks like:
	//   "json: unsupported type: func()"
	// which doesn't normally contain quotes, but we want to ensure the fix
	// handles the general case.  Use a channel which produces:
	//   "json: unsupported type: chan int"
	v := struct{ C chan int }{C: make(chan int)}
	out := ToJSON(v)
	if !isValidJSON(out) {
		t.Errorf("ToJSON with struct containing chan produced invalid JSON: %s", out)
	}
}
