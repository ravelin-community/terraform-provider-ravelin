package ravelinaccess

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestExtractRavelinAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		file     string
		expected RavelinAccess
		user     RavelinAccess
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
				GCP: GCPAccess{
					Groups: []string{"group"},
				},
				Gsudo: GsudoAccess{
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
				GCP: GCPAccess{
					Groups: []string{"group"},
				},
				Gsudo: GsudoAccess{
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
				GCP: GCPAccess{
					Groups: []string{"group"},
				},
				Gsudo: GsudoAccess{
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
				GCP: GCPAccess{
					Groups: []string{"group", "another-group"},
				},
				Gsudo: GsudoAccess{
					Inherit:     false,
					Escalations: map[string][]string{"test-user-project": {"roles/owner", "roles/editor"}},
				},
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.extractAccess([]byte(tt.input))
			if tt.expError == "" {
				require.NoError(t, err)
			}
			if err != nil {
				require.ErrorContains(t, err, tt.expError)
			}

			if diff := cmp.Diff(tt.user, tt.expected); diff != "" {
				t.Errorf("expected ravelin access data (+) but got (-), %s", diff)
			}

		})
	}
}

func TestExpandCustomRoles(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string][]string
		expOutput map[string][]string
	}{
		{
			name:      "no_custom_roles",
			input:     map[string][]string{"test-project": {"roles/owner", "role/editor"}},
			expOutput: map[string][]string{"test-project": {"roles/owner", "role/editor"}},
		},
		{
			name: "expand_custom_role",
			input: map[string][]string{
				"test-project":  {"roles/owner", "custom/admin"},
				"test-project2": {"roles/bigquery.admin", "custom/editor"},
			},
			expOutput: map[string][]string{
				"test-project":  {"roles/owner", "projects/test-project/roles/admin"},
				"test-project2": {"roles/bigquery.admin", "projects/test-project2/roles/editor"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := expandCustomRoles(tt.input)

			if diff := cmp.Diff(out, tt.expOutput); diff != "" {
				t.Errorf("transfromCustomRoles - expected (+) but got (-), %s", diff)
			}
		})
	}
}
