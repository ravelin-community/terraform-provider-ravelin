package ravelinaccess

import (
	"errors"
	"fmt"
	"path/filepath"
)

// TwingateAccess represents the Twingate access configuration for a user or a group.
type TwingateAccess struct {
	// Enabled indicates if the user has Twingate access. Please note that this is
	// a pointer as we might want to have "false" over a user override a group
	// level access.
	Enabled *bool `yaml:"enabled,omitempty"`
	// Admin indicates if the user has Twingate admin access. Please note that this is
	// a pointer as we might want to have "false" over a user override a group
	// level access.
	Admin *bool `yaml:"admin,omitempty"`
}

// InheritTwingateAccess inherits the Twingate access from the primary group.
// Any setting at the user level will override the group level setting.
func (a *RavelinAccess) InheritTwingateAccess() error {
	if a.Type != USER {
		return errors.New("inheritance is only available for users")
	}

	// if we have no groups, we have nothing to do
	if len(a.GCP.Groups) == 0 {
		return nil
	}

	// we only inherit twingate access through the primary group
	primaryGroupFile := a.GCP.Groups[0] + ".yml"

	groupFile := filepath.Join(filepath.Dir(a.filePath), "..", "groups", primaryGroupFile)

	data, err := ReadFile(groupFile)
	if err != nil {
		return fmt.Errorf("error reading group file %s: %w", groupFile, err)
	}

	var group RavelinAccess
	if err := group.extractAccess(data); err != nil {
		return fmt.Errorf("error extracting group access: %w", err)
	}

	// inherit twingate access from the group only if the user level settings are
	// not set
	if group.Twingate.Enabled != nil && a.Twingate.Enabled == nil {
		a.Twingate.Enabled = group.Twingate.Enabled
	}

	if *a.Twingate.Enabled && group.Twingate.Admin != nil && a.Twingate.Admin == nil {
		a.Twingate.Admin = group.Twingate.Admin
	}

	return nil
}
