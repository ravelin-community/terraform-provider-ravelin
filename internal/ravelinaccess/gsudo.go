package ravelinaccess

import (
	"errors"
	"fmt"
	"path/filepath"
)

// GsudoAccess represents the gsudo configuration for a user or a group.
type GsudoAccess struct {
	// Escalations is a map of project names to a list of escalation roles.
	Escalations map[string][]string `yaml:"escalations"`
	// Inherit indicates if escalations are inherited from the user's group. Inheritance is only
	// supported for users.
	Inherit bool `yaml:"inherit"`
}

// InheritGsudoAccess inherits the gsudo escalations from the list of groups the user belongs to.
func (a *RavelinAccess) InheritGsudoAccess() error {
	if a.Type != USER {
		return errors.New("inheritance is only available for users")
	}

	// nothing to do if we don't want to inherit group level escalations
	if !a.Gsudo.Inherit {
		return nil
	}

	for _, group := range a.GCP.Groups {
		groupFile := filepath.Join(filepath.Dir(a.filePath), "..", "groups", group+".yml")
		groupYaml, err := readFile(groupFile)
		if err != nil {
			return fmt.Errorf("error reading group file %s: %w", groupFile, err)
		}

		var group RavelinAccess
		if err := group.extractAccess(groupYaml); err != nil {
			return fmt.Errorf("error extracting group access: %w", err)
		}

		a.Gsudo.Escalations = mergeMapsOfSlices(a.Gsudo.Escalations, group.Gsudo.Escalations)
	}

	return nil
}
