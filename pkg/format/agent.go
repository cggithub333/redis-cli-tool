package format

import (
	"encoding/json"
	"io"
	"log"
	"strings"
)

// PurifyStream suppresses standard loggers to guarantee clean stdout for JSON consumers
func PurifyStream() {
	log.SetOutput(io.Discard)
}

// FilterFields extracts only specified fields from an object, returning a filtered map
func FilterFields(item interface{}, allowedFields []string) map[string]interface{} {
	if len(allowedFields) == 0 {
		var m map[string]interface{}
		data, _ := json.Marshal(item)
		_ = json.Unmarshal(data, &m)
		return m
	}

	fieldSet := make(map[string]bool)
	for _, f := range allowedFields {
		clean := strings.ToLower(strings.TrimSpace(f))
		if clean != "" {
			fieldSet[clean] = true
		}
	}

	var rawMap map[string]interface{}
	data, _ := json.Marshal(item)
	_ = json.Unmarshal(data, &rawMap)

	filtered := make(map[string]interface{})
	for k, v := range rawMap {
		if fieldSet[strings.ToLower(k)] {
			filtered[k] = v
		}
	}
	return filtered
}

// SerializeJSON formats an object as either single-line compact JSON or indented JSON
func SerializeJSON(data interface{}, compact bool) (string, error) {
	if compact {
		b, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
