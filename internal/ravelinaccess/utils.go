package ravelinaccess

import "slices"

// mergeMapsOfSlices merges two maps of string slices into a new map of string slices.
// It returns a new map with all keys from both maps, and the values are the
// concatenation of the values from both maps, deduplicated.
func mergeMapsOfSlices(m1, m2 map[string][]string) map[string][]string {
	merged := make(map[string][]string, len(m1)+len(m2))

	allKeys := make(map[string]struct{})
	for k := range m1 {
		allKeys[k] = struct{}{}
	}
	for k := range m2 {
		allKeys[k] = struct{}{}
	}

	for k := range allKeys {
		var combined []string
		if v1, ok := m1[k]; ok {
			combined = append(combined, v1...)
		}
		if v2, ok := m2[k]; ok {
			combined = append(combined, v2...)
		}
		merged[k] = dedupSlice(combined)
	}

	return merged
}

// dedupSlice deduplicates a slice of strings.
func dedupSlice(s []string) []string {
	slices.Sort(s)
	return slices.Compact(s)
}
