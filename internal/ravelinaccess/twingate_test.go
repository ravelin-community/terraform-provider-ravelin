package ravelinaccess

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// createTempFiles creates a temporary directory with the given files.
func createTempFiles(t *testing.T, files ...map[string][]byte) string {
	tempDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tempDir, "users"), 0755); err != nil {
		t.Fatalf("error creating users directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "groups"), 0755); err != nil {
		t.Fatalf("error creating groups directory: %v", err)
	}

	for _, typ := range files {
		for path, data := range typ {
			err := os.WriteFile(filepath.Join(tempDir, path), data, 0644)
			if err != nil {
				t.Fatalf("error writing file: %v", err)
			}
		}
	}
	return tempDir
}

func TestInheritTwingateAccess(t *testing.T) {
	tests := []struct {
		name       string
		userFile   map[string][]byte
		groupFiles map[string][]byte
		expEnabled bool
		expAdmin   bool
		expErr     bool
	}{
		{
			name: "user_no_groups",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
twingate:
  enabled: true
  admin: true
`),
			},
			groupFiles: map[string][]byte{},
			expEnabled: true,
			expAdmin:   true,
		},
		{
			name: "user_with_groups",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
gcp:
  groups:
    - group1
`),
			},
			groupFiles: map[string][]byte{
				"groups/group1.yml": []byte(`
twingate:
  enabled: true
  admin: true
`),
			},
			expEnabled: true,
			expAdmin:   true,
		},
		{
			name: "user_with_groups_no_inheritance",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
twingate:
  enabled: false`),
			},
			groupFiles: map[string][]byte{
				"groups/group1.yml": []byte(`
twingate:
  enabled: true
  admin: true
`),
			},
			expEnabled: false, // overriden by the user level setting
			expAdmin:   false, // can't be an admin if not enabled
		},
		{
			name: "user_with_groups_no_admin",
			userFile: map[string][]byte{
				"users/john_doe.yml": []byte(`
twingate:
  admin: false`),
			},
			groupFiles: map[string][]byte{
				"groups/group1.yml": []byte(`
twingate:
  enabled: true
  admin: true
`),
			},
			expEnabled: true,
			expAdmin:   false, // overriden by the user level setting
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := createTempFiles(t, tt.userFile, tt.groupFiles)

			access, err := ExtractRavelinAccess(filepath.Join(tempDir, "users", "john_doe.yml"))
			if err != nil {
				t.Fatalf("error extracting user access: %v", err)
			}

			err = access.InheritTwingateAccess()
			if !tt.expErr {
				require.NoError(t, err)
			}

			if access.Twingate.Enabled != nil && *access.Twingate.Enabled != tt.expEnabled {
				t.Fatalf("want enabled: %t, got: %t", tt.expEnabled, *access.Twingate.Enabled)
			}

			if access.Twingate.Admin != nil && *access.Twingate.Admin != tt.expAdmin {
				t.Fatalf("want admin: %t, got: %t", tt.expAdmin, *access.Twingate.Admin)
			}
		})
	}
}
