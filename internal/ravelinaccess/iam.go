package ravelinaccess

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// RavelinAccess can be assigned to an entity which can either be a user, a service
// or a group.
type EntityType int32

const (
	USER    EntityType = 0
	GROUP   EntityType = 1
	SERVICE EntityType = 2
)

// RavelinAccess represents the access configuration at Ravelin, it can describe
// the access to multiple services and platforms for a user or a workspace group.
type RavelinAccess struct {
	// Email is the email of the user or group.
	Email string
	// Type indicates if the ravelin access is assigned to a user, a group or a service.
	Type EntityType

	// GCP represents the GCP IAM roles and groups for the user or group. For now it only supports
	// the groups. Only users can be part of groups, groups cannot be part of other groups.
	GCP GCPAccess `yaml:"gcp,omitempty"`
	// Gsudo represents the gsudo configuration for the user or group.
	Gsudo GsudoAccess `yaml:"gsudo,omitempty"`
	// Twingate represents the Twingate access configuration for the user or group.
	Twingate TwingateAccess `yaml:"twingate,omitempty"`

	// Keeping track of original file
	filePath string
}

// GCPAccess represents the GCP IAM roles and groups for a user or a group.
type GCPAccess struct {
	// Groups is a list of google workspace groups the user belongs to.
	Groups []string `yaml:"groups,omitempty"`
}

func ExtractRavelinAccess(filePath string) (RavelinAccess, error) {
	data, err := readFile(filePath)
	if err != nil {
		return RavelinAccess{}, fmt.Errorf("error reading file: %w", err)
	}

	var acc RavelinAccess
	acc.filePath = filePath

	acc.Type, err = fileToType(filePath)
	if err != nil {
		return RavelinAccess{}, fmt.Errorf("error determining type of entity from file: %w", err)
	}

	acc.Email, err = fileToEmail(filePath, acc.Type)
	if err != nil {
		return RavelinAccess{}, fmt.Errorf("error determining email of file: %w", err)
	}

	err = acc.extractAccess(data)
	if err != nil {
		return RavelinAccess{}, fmt.Errorf("error extracting user access: %w", err)
	}

	return acc, nil
}

func (a *RavelinAccess) extractAccess(data []byte) error {
	if err := yaml.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("error unmarshaling IAM file: %w", err)
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
