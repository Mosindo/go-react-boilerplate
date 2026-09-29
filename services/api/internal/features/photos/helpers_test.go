package photos_test

import "encoding/json"

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

func flip(c byte) string {
	if c == '0' {
		return "1"
	}
	return "0"
}
