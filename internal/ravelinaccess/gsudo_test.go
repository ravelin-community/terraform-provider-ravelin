package ravelinaccess

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestInheritGsudoAccess(t *testing.T) {
	tests := []struct {
		name       string
		userFile   map[string][]byte
		groupFiles map[string][]byte
		expected   GsudoAccess
		expError   string
	}{
		{
			name: "normal_usage",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
gcp:
  groups:
    - group1
gsudo:
  inherit: true
  escalations:
    project1:
      - roles/owner
`)},
			groupFiles: map[string][]byte{
				"groups/group1.yml": []byte(`
gsudo:
  escalations:
    project2:
      - roles/owner
`)},
			expected: GsudoAccess{
				Inherit: true,
				Escalations: map[string][]string{
					"project1": {"roles/owner"},
					"project2": {"roles/owner"},
				},
			},
		},
		{
			name: "no_inherit",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
gcp:
  groups:
    - group1
gsudo:
  inherit: false
  escalations:
    project1:
      - roles/owner
`)},
			groupFiles: map[string][]byte{
				"groups/group1.yml": []byte(`
gsudo:
  escalations:
    project2:
      - roles/owner
`)},
			expected: GsudoAccess{
				Inherit: false,
				Escalations: map[string][]string{
					"project1": {"roles/owner"},
				},
			},
		},
		{
			name: "multiple groups",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
gcp:
  groups:
    - group1
    - group2
gsudo:
  inherit: true
  escalations:
    project1:
      - roles/owner
      - roles/editor
`)},
			groupFiles: map[string][]byte{
				"groups/group1.yml": []byte(`
gsudo:
 escalations:
  project2:
    - roles/owner
`),
				"groups/group2.yml": []byte(`
gsudo:
  escalations:
    project1:
      - roles/bigquery.admin
`)},
			expected: GsudoAccess{
				Inherit: true,
				Escalations: map[string][]string{
					"project1": {"roles/bigquery.admin", "roles/editor", "roles/owner"},
					"project2": {"roles/owner"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := createTempFiles(t, tt.userFile, tt.groupFiles)

			access, err := ExtractRavelinAccess(filepath.Join(tempDir, "users", "john_doe.yml"))
			if err != nil {
				t.Fatalf("error extracting user access: %v", err)
			}

			err = access.InheritGsudoAccess()
			if tt.expError == "" {
				require.NoError(t, err)
			}
			if err != nil {
				require.ErrorContains(t, err, tt.expError)
			}

			if diff := cmp.Diff(access.Gsudo, tt.expected); diff != "" {
				t.Errorf("expected ravelin access data (+) but got (-), %s", diff)
			}
		})
	}
}
