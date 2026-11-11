package support

import (
	"bytes"
)

func EnvironmentMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		key, item, ok := bytes.Cut([]byte(value), []byte("="))
		if ok {
			result[string(key)] = string(item)
		}
	}
	return result
}
