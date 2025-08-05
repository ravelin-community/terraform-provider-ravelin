package ravelinaccess

import "testing"

func TestFileToType(t *testing.T) {
	tests := []struct {
		file    string
		want    EntityType
		wantErr bool
	}{
		{file: "users/john_doe.yml", want: USER},
		{file: "service-accounts/john_doe.yml", want: SERVICE},
		{file: "groups/john_doe.yml", want: GROUP},
		{file: "path/users/john_doe.yml", want: USER},
		{file: "path/service-accounts/john_doe.yml", want: SERVICE},
		{file: "path/groups/john_doe.yml", want: GROUP},
		{file: "path/john_doe.yml", want: -1, wantErr: true},
	}

	for _, test := range tests {
		got, err := fileToType(test.file)
		if (err != nil) != test.wantErr {
			t.Errorf("fileToType(%q) error = %v, wantErr %v", test.file, err, test.wantErr)
		}
		if got != test.want {
			t.Errorf("fileToType(%q) = %v, want %v", test.file, got, test.want)
		}
	}
}

func TestUserFileToEmail(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		typ       EntityType
		expOutput string
	}{
		{
			name:      "user_file",
			input:     "/mnt/c/iam/users/john_doe.yaml",
			typ:       USER,
			expOutput: "john.doe@ravelin.com",
		},
		{
			name:      "check_hyphens_in_name",
			input:     "../../iam/users/marie-josette_doe.yaml",
			typ:       USER,
			expOutput: "marie-josette.doe@ravelin.com",
		},
		{
			name:      "group_file",
			input:     "/mnt/c/iam/groups/group_name.yaml",
			typ:       GROUP,
			expOutput: "gcp-group_name@ravelin.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := fileToEmail(tt.input, tt.typ)
			if err != nil {
				t.Errorf("fileToEmail - error: %v", err)
			}
			if out != tt.expOutput {
				t.Errorf("fileToEmail - expected %s but got %s", tt.expOutput, out)
			}
		})
	}
}
