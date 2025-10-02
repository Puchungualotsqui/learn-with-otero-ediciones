package helper

import (
	"fmt"
	"strconv"
	"strings"
)

func StringsToInts(ss ...string) ([]int, error) {
	out := make([]int, len(ss))
	for i, s := range ss {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, fmt.Errorf("invalid int at index %d (%q): %w", i, s, err)
		}
		out[i] = n
	}
	return out, nil
}
