package ravelinaccess

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestUserFileToEmail(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expOutput string
	}{
		{
			name:      "normal usage",
			input:     "/mnt/c/iam/users/john_doe.yaml",
			expOutput: "john.doe@ravelin.com",
		},
		{
			name:      "check hyphens in name",
			input:     "../../iam/users/marie-josette_doe.yaml",
			expOutput: "marie-josette.doe@ravelin.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := userFileToEmail(tt.input)
			if out != tt.expOutput {
				t.Errorf("fileToEmail - expected %s but got %s", tt.expOutput, out)
			}
		})
	}
}

func TestExtractRavelinAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		file     string
		expected RavelinAccess
		expError string
	}{
		{
			name: "normal usage",
			input: `
gcp:
  groups:
    - group
gsudo:
  inherit: true
  escalations:
    test-user-project:
      - roles/owner
`,
			expected: RavelinAccess{
				GCPAccess: GCPAccess{
					Groups: []string{"group"},
				},
				GsudoAccess: GsudoAccess{
					Inherit:     true,
					Escalations: map[string][]string{"test-user-project": {"roles/owner"}},
				},
			}},
		{
			name: "false inherit",
			input: `
gcp:
  groups:
    - group
gsudo:
  inherit: false
  escalations:
    test-user-project:
      - roles/owner
`,
			expected: RavelinAccess{
				GCPAccess: GCPAccess{
					Groups: []string{"group"},
				},
				GsudoAccess: GsudoAccess{
					Inherit:     false,
					Escalations: map[string][]string{"test-user-project": {"roles/owner"}},
				},
			}},
		{
			name: "multiple escalations",
			input: `
gcp:
  groups:
    - group
gsudo:
  inherit: false
  escalations:
    test-user-project:
      - roles/owner
      - roles/editor
    test-project:
      - roles/editor
`,
			expected: RavelinAccess{
				GCPAccess: GCPAccess{
					Groups: []string{"group"},
				},
				GsudoAccess: GsudoAccess{
					Inherit: false,
					Escalations: map[string][]string{"test-user-project": {"roles/owner", "roles/editor"},
						"test-project": {"roles/editor"}},
				},
			}},
		{
			name: "multiple groups",
			input: `
gcp:
  groups:
    - group
    - another-group
gsudo:
  inherit: false
  escalations:
    test-user-project:
      - roles/owner
      - roles/editor
`,
			expected: RavelinAccess{
				GCPAccess: GCPAccess{
					Groups: []string{"group", "another-group"},
				},
				GsudoAccess: GsudoAccess{
					Inherit:     false,
					Escalations: map[string][]string{"test-user-project": {"roles/owner", "roles/editor"}},
				},
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, out := exctractAccess([]byte(tt.input))
			if tt.expError == "" {
				require.NoError(t, err)
			}
			if err != nil {
				require.ErrorContains(t, err, tt.expError)
			}

			if diff := cmp.Diff(out, tt.expected); diff != "" {
				t.Errorf("expected ravelin access data (+) but got (-), %s", diff)
			}

		})
	}
}
