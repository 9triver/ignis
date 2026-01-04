package benchmarks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func generateObject(n int) string {
	return strings.Repeat("b", n)
}

var results []map[string]any

func WriteResult(kvs ...any) {
	result := make(map[string]any)
	var key string
	for i, korv := range kvs {
		if i%2 == 0 {
			key = korv.(string)
		} else {
			result[key] = korv
		}
	}

	results = append(results, result)
}

func SaveResult(expr string) {
	data, _ := json.MarshalIndent(results, "", "  ")
	out := fmt.Sprintf("output-%s.json", expr)
	os.WriteFile(out, data, 0644)

	// reset results for next experiment
	results = nil
}
