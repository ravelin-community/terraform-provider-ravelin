package ravelinaccess

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gopkg.in/yaml.v3"
)

type RavelinAccess struct {
	Email    string
	GCP      GCPAccess      `yaml:"gcp,omitempty"`      // GCP IAM roles and groups
	Gsudo    GsudoAccess    `yaml:"gsudo,omitempty"`    // gsudo configuration for the user
	Twingate TwingateAccess `yaml:"twingate,omitempty"` // Twingate access configuration for the user
}

type GsudoAccess struct {
	Escalations map[string][]string `yaml:"escalations"` // list of escalation roles per project
	Inherit     bool                `yaml:"inherit"`     // whether the roles are inherited from a group
}

type GCPAccess struct {
	Groups []string `yaml:"groups,omitempty"` // list of google workspace groups the user belongs to
}

type TwingateAccess struct {
	Enabled bool `yaml:"enabled"` // whether the user has Twingate access
	Admin   bool `yaml:"admin"`   // whether the user has Twingate admin access
}

func ExtractUserAccess(ctx context.Context, iamDirectory string) ([]RavelinAccess, error) {
	users := make([]RavelinAccess, 0, 200) // Preallocate slice for 200 users
	err, userFiles := getUserFiles(iamDirectory)
	if err != nil {
		return nil, fmt.Errorf("error getting user files:  %v", err)
	}

	for _, userFile := range userFiles {
		if !strings.HasSuffix(userFile, ".yml") {
			tflog.Info(ctx, fmt.Sprintf("Skipping non-YAML file: %s", userFile))
			continue
		}

		yaml, err := readYamlFile(fmt.Sprintf("%s/users/%s", iamDirectory, userFile))
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("error reading user file %s: %v", userFile, err))
		}
		if len(yaml) == 0 {
			tflog.Info(ctx, fmt.Sprintf("Skipping empty user file: %s", userFile))
			continue
		}

		user, err := exctractAccess(yaml)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("error extracting user access: %v", err))
		}

		user.Email = userFileToEmail(userFile)

		for i, g := range user.GCP.Groups {
			yaml, err := readYamlFile(fmt.Sprintf("%s/groups/%s.yml", iamDirectory, g))
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("error reading group file %s: %v", g, err))
			}
			if len(yaml) == 0 {
				tflog.Info(ctx, fmt.Sprintf("Skipping empty group file: %s", g))
				continue
			}

			group, err := exctractAccess(yaml)
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("error extracting group access for %s: %v", g, err))
			}
			if user.GCP.Groups != nil && user.Gsudo.Inherit && i == 0 {
				// Users inherit escalations from the first group only
				user.Gsudo.Escalations = MergeMapsOfSlices(user.Gsudo.Escalations, group.Gsudo.Escalations)
			}
		}

		users = append(users, user)
	}
	return users, nil
}

func getUserFiles(iamDirectory string) (error, []string) {
	files, err := os.ReadDir(fmt.Sprintf("%s/users", iamDirectory))
	if err != nil {
		return fmt.Errorf("error retrieving a list of user files from IAM directory: %v", err), nil
	}

	var userFiles []string
	for _, file := range files {
		if !file.IsDir() {
			userFiles = append(userFiles, file.Name())
		}
	}

	return nil, userFiles
}

func readYamlFile(filePath string) ([]byte, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file %s: %v", filePath, err)
	}
	if len(yamlFile) == 0 {
		return nil, fmt.Errorf("YAML file %s is empty", filePath)
	}
	return yamlFile, nil
}

func exctractAccess(data []byte) (RavelinAccess, error) {
	var access RavelinAccess

	if err := yaml.Unmarshal(data, &access); err != nil {
		return access, fmt.Errorf("error unmarshaling IAM file: %v", err)
	}

	if access.Gsudo.Escalations == nil {
		access.Gsudo.Escalations = make(map[string][]string)
	}

	if access.GCP.Groups == nil {
		access.GCP.Groups = make([]string, 0)
	}

	// Ensure custom roles are transformed to full GCP role names
	access.Gsudo.Escalations = transformCustomRoles(access.Gsudo.Escalations)

	return access, nil
}

func userFileToEmail(file string) string {
	return strings.ReplaceAll(strings.Split(filepath.Base(file), ".")[0], "_", ".") + "@ravelin.com"
}

// Alternative to maps.Copy when overwriting existing keys is not desired.
func MergeMapsOfSlices[K comparable](dst, src map[K][]string) map[K][]string {
	merged := make(map[K][]string, len(dst)+len(src))
	for k, v := range dst {
		merged[k] = dedupSlices(slices.Clone(v))
	}
	for k, vSrc := range src {
		if vDst, exists := merged[k]; exists {
			merged[k] = dedupSlices(slices.Concat(vDst, vSrc))
		} else {
			merged[k] = dedupSlices(slices.Clone(vSrc))
		}
	}
	return merged
}

func dedupSlices(s []string) []string {
	seen := make(map[string]struct{})
	var result []string
	result = make([]string, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func transformCustomRoles(m map[string][]string) map[string][]string {
	for p, roles := range m {
		for i, role := range roles {
			if strings.HasPrefix(role, "custom/") {
				roles[i] = fmt.Sprintf("projects/%s/roles%s", p, strings.Trim(role, "custom"))
			}
		}
	}
	return m
}
