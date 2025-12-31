package helper

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// NormalizeFilename ensures only safe characters remain
func NormalizeFilename(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)

	// replace spaces with underscores
	base = strings.ReplaceAll(base, " ", "_")

	// remove any characters not alphanumeric, dash or underscore
	re := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	base = re.ReplaceAllString(base, "")

	// lowercase for consistency
	base = strings.ToLower(base)

	return base + ext
}

func IntsToStrings(ii ...int) []string {
	out := make([]string, len(ii))
	for i, n := range ii {
		out[i] = strconv.Itoa(n)
	}
	return out
}

func UniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
