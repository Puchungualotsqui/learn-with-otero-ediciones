package helper

import "encoding/json"

func MustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
