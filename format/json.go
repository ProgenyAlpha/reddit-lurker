package format

import (
	"encoding/json"
)

// ToJSON serializes any value to indented JSON.
func ToJSON(v any) string {
	if v == nil {
		return "{}"
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	return string(b)
}
