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
		// Use json.Marshal on the error string so quotes/backslashes are escaped properly.
		errMsg, _ := json.Marshal(err.Error())
		return `{"error": ` + string(errMsg) + `}`
	}
	return string(b)
}
