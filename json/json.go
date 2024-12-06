package json

import "encoding/json"

func SafeMarshalJson(v any) string {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(jsonBytes)
}
