package agent

import "encoding/json"

// ParseLine attempts to extract human-readable text from a JSON line.
// If the line looks like JSON, it tries common field names. Otherwise it
// returns the line unchanged.
func ParseLine(line string) string {
	if len(line) == 0 || line[0] != '{' {
		return line
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return line
	}

	// Try common Claude output-format JSON fields in priority order
	for _, field := range []string{"content", "text", "message"} {
		if v, ok := obj[field]; ok {
			switch val := v.(type) {
			case string:
				if val != "" {
					return val
				}
			}
		}
	}

	return line
}
