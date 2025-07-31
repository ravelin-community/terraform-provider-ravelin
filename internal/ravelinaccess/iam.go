package ravelinaccess

import (
	"fmt"
	"path/filepath"
	"strings"

	"slices"

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
	Enabled *bool `yaml:"enabled,omitempty"` // whether the user has Twingate access
	Admin   *bool `yaml:"admin,omitempty"`   // whether the user has Twingate admin access
}

func ExtractEntityAccess(yaml []byte, fileName string) (RavelinAccess, error) {

	var user RavelinAccess
	err := user.extractAccess(yaml)
	if err != nil {
		return RavelinAccess{}, fmt.Errorf("error extracting user access: %v", err)
	}

	user.Email = userFileToEmail(fileName)
	return user, nil
}

func (user *RavelinAccess) InheritGroupEscalations(groupYamls map[int][]byte) error {
	for _, groupYaml := range groupYamls {
		var group RavelinAccess
		if err := group.extractAccess(groupYaml); err != nil {
			return fmt.Errorf("error extracting group access: %v", err)
		}
		if user.Gsudo.Inherit {
			user.Gsudo.Escalations = MergeMapsOfSlices(user.Gsudo.Escalations, group.Gsudo.Escalations)
		}

	}
	return nil
}

func (a *RavelinAccess) extractAccess(data []byte) error {

	if err := yaml.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("error unmarshaling IAM file: %v", err)
	}

	if a.Gsudo.Escalations == nil {
		a.Gsudo.Escalations = make(map[string][]string)
	}

	if a.GCP.Groups == nil {
		a.GCP.Groups = make([]string, 0)
	}

	// Ensure custom roles are transformed to full GCP role names
	a.Gsudo.Escalations = transformCustomRoles(a.Gsudo.Escalations)

	return nil
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
