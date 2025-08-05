package ravelinaccess

import (
	"reflect"
	"testing"
)

func TestMergeMapsOfSlices(t *testing.T) {
	tests := []struct {
		name      string
		dst       map[string][]string
		src       map[string][]string
		expOutput map[string][]string
	}{
		{
			name: "merge_maps_of_slices",
			dst: map[string][]string{
				"a": {"1", "2", "5"},
				"b": {"3", "4"},
			},
			src: map[string][]string{
				"a": {"1", "2", "3"},
				"b": {"3", "4", "5"},
			},
			expOutput: map[string][]string{
				"a": {"1", "2", "3", "5"},
				"b": {"3", "4", "5"},
			},
		},
		{
			name:      "merge_maps_of_slices_with_nil",
			dst:       nil,
			src:       map[string][]string{"a": {"1", "2", "3"}},
			expOutput: map[string][]string{"a": {"1", "2", "3"}},
		},
		{
			name:      "merge_maps_of_slices_with_nil_src",
			dst:       map[string][]string{"a": {"1", "2", "3"}},
			src:       nil,
			expOutput: map[string][]string{"a": {"1", "2", "3"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := mergeMapsOfSlices(tt.dst, tt.src)
			if !reflect.DeepEqual(out, tt.expOutput) {
				t.Errorf("mergeMapsOfSlices - expected %s but got %s", tt.expOutput, out)
			}
		})
	}
}
