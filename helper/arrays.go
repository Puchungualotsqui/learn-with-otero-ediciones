package helper

import "slices"

// Contains returns true if elem is in slice.
// This is just a thin wrapper around slices.Contains for clarity.
func Contains[T comparable](slice []T, elem T) bool {
	return slices.Contains(slice, elem)
}

func Remove[T comparable](slice []T, id T) []T {
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if v != id {
			result = append(result, v)
		}
	}
	return result
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func First[T any](v []*T) *T {
	if len(v) == 0 {
		return nil
	}
	return v[0]
}
