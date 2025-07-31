package ravelinaccess

import (
	"reflect"
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

func TestGsudoInheritance(t *testing.T) {
	tests := []struct {
		name       string
		userInput  string
		groupInput map[int][]byte
		file       string
		expected   RavelinAccess
		user       RavelinAccess
		expError   string
	}{
		{
			name: "normal usage",
			userInput: `
gcp:
  groups:
    - group
gsudo:
  inherit: true
  escalations:
    test-user-project:
      - roles/owner
`,
			groupInput: map[int][]byte{
				0: []byte(`
gcp: {}
gsudo:
  escalations:
    test-group-project:
      - roles/owner
`),
			},
			expected: RavelinAccess{
				GCP: GCPAccess{
					Groups: []string{"group"},
				},
				Gsudo: GsudoAccess{
					Inherit: true,
					Escalations: map[string][]string{
						"test-user-project":  {"roles/owner"},
						"test-group-project": {"roles/owner"},
					},
				},
			}},
		{
			name: "false inherit",
			userInput: `
gcp:
  groups:
    - group
gsudo:
  inherit: false
  escalations:
    test-user-project:
      - roles/owner
`,
			groupInput: map[int][]byte{
				0: []byte(`
gcp: {}
gsudo:
  escalations:
    test-group-project:
      - roles/owner
`),
			},
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
			name: "multiple groups",
			userInput: `
gcp:
  groups:
    - group
    - another-group
gsudo:
  inherit: true
  escalations:
    test-user-project:
      - roles/owner
      - roles/editor
`,
			groupInput: map[int][]byte{
				0: []byte(`
gcp: {}
gsudo:
  escalations:
    test-group-project:
      - roles/owner
`),
				1: []byte(`
gcp: {}
gsudo:
  escalations:
    test-group2-project:
      - roles/editor
`),
			},
			expected: RavelinAccess{
				GCP: GCPAccess{
					Groups: []string{"group", "another-group"},
				},
				Gsudo: GsudoAccess{
					Inherit:     true,
					Escalations: map[string][]string{"test-user-project": {"roles/owner", "roles/editor"}, "test-group-project": {"roles/owner"}, "test-group2-project": {"roles/editor"}},
				},
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.extractAccess([]byte(tt.userInput))
			if tt.expError == "" {
				require.NoError(t, err)
			}
			if err != nil {
				require.ErrorContains(t, err, tt.expError)
			}

			err = tt.user.InheritGroupEscalations(tt.groupInput)
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
			if !reflect.DeepEqual(out, tt.expOutput) {
				t.Errorf("transfromCustomRoles - expected %s but got %s", tt.expOutput, out)
			}
		})
	}
}
