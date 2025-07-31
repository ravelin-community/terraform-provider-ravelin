package ravelinaccess

import (
	"fmt"
	"path/filepath"
	"strings"

	"slices"

	"gopkg.in/yaml.v3"
)

// RavelinAccess represents the access configuration at Ravelin, it can describe
// the access to multiple services and platforms for a user or a workspace group.
type RavelinAccess struct {
	// Email is the email of the user or group.
	Email string
	// IsGroup indicates if the ravelin access is for a group or a user.
	IsGroup bool

	// GCP represents the GCP IAM roles and groups for the user or group. For now it only supports
	// the groups. Only users can be part of groups, groups cannot be part of other groups.
	GCP GCPAccess `yaml:"gcp,omitempty"`
	// Gsudo represents the gsudo configuration for the user or group.
	Gsudo GsudoAccess `yaml:"gsudo,omitempty"`
	// Twingate represents the Twingate access configuration for the user or group.
	Twingate TwingateAccess `yaml:"twingate,omitempty"`
}

// GsudoAccess represents the gsudo configuration for a user or a group.
type GsudoAccess struct {
	// Escalations is a map of project names to a list of escalation roles.
	Escalations map[string][]string `yaml:"escalations"`
	// Inherit indicates if escalations are inherited from the user's group. Inheritance is only
	// supported for users.
	Inherit bool `yaml:"inherit"`
}

// GCPAccess represents the GCP IAM roles and groups for a user or a group.
type GCPAccess struct {
	// Groups is a list of google workspace groups the user belongs to.
	Groups []string `yaml:"groups,omitempty"`
}

// TwingateAccess represents the Twingate access configuration for a user or a group.
type TwingateAccess struct {
	// Enabled indicates if the user has Twingate access.
	Enabled *bool `yaml:"enabled,omitempty"`
	// Admin indicates if the user has Twingate admin access.
	Admin *bool `yaml:"admin,omitempty"`
}

func ExtractEntityRavelinAccess(yaml []byte, fileName string) (RavelinAccess, error) {
	var acc RavelinAccess
	err := acc.extractAccess(yaml)
	if err != nil {
		return RavelinAccess{}, fmt.Errorf("error extracting user access: %v", err)
	}

	acc.Email = userFileToEmail(fileName)
	return acc, nil
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
	a.Gsudo.Escalations = expandCustomRoles(a.Gsudo.Escalations)

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

// expandCustomRoles expands custom roles from their short form to the full GCP
// project level custom role reference. It takes as input a map or project to a
// list of roles and returns the same map with the custom roles expanded.
func expandCustomRoles(m map[string][]string) map[string][]string {
	for project, roles := range m {
		for i, role := range roles {
			if strings.HasPrefix(role, "custom/") {
				roles[i] = fmt.Sprintf("projects/%s/roles/%s", project, role[7:])
			}
		}
	}
	return m
}
