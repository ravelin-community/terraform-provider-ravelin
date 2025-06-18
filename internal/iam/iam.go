package RavelinAccess

import (
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type RavelinAccess struct {
	Email       string
	GCPAccess   GCPAccess   `yaml:"gcp,omitempty"`   // GCP IAM roles and groups
	GsudoAccess GsudoAccess `yaml:"gsudo,omitempty"` // gsudo configuration for the user
}

type GsudoAccess struct {
	Escalations map[string][]string `yaml:"escalations"` // list of escalation roles per project
	Inherit     bool                `yaml:"inherit"`     // whether the roles are inherited from a group
}

type GCPAccess struct {
	Groups []string `yaml:"groups,omitempty"` // list of google workspace groups the user belongs to
}

func ExtractUserAccess(iamDirectory string) (error, []RavelinAccess) {
	users := make([]RavelinAccess, 0)
	err, userFiles := getUserFiles(iamDirectory)
	if err != nil {
		return fmt.Errorf("error getting user files:  %v", err), nil
	}

	for _, userFile := range userFiles {
		if !strings.HasSuffix(userFile, ".yaml") {
			log.Printf("Skipping non-YAML file: %s", userFile)
			continue
		}

		err, yaml := readYamlFile(fmt.Sprintf("%s/users/%s", iamDirectory, userFile))
		if err != nil {
			log.Fatalf("error reading user file %s: %v", userFile, err)
		}
		if len(yaml) == 0 {
			log.Printf("Skipping empty user file: %s", userFile)
			continue
		}

		err, user := exctractAccess(yaml)
		if err != nil {
			log.Fatalf("error extracting user access: %v", err)
		}

		user.Email = userFileToEmail(userFile)

		for _, g := range user.GCPAccess.Groups {
			err, yaml := readYamlFile(fmt.Sprintf("%s/groups/%s.yaml", iamDirectory, g))
			if err != nil {
				log.Fatalf("error reading group file %s: %v", g, err)
			}
			if len(yaml) == 0 {
				log.Printf("Skipping empty group file: %s", g)
				continue
			}

			err, group := exctractAccess(yaml)
			if err != nil {
				log.Fatalf("error extracting group access for %s: %v", g, err)
			}

			maps.Copy(user.GsudoAccess.Escalations, group.GsudoAccess.Escalations)
		}

		users = append(users, user)
	}
	return nil, users
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

func readYamlFile(filePath string) (error, []byte) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading YAML file %s: %v", filePath, err), nil
	}
	if len(yamlFile) == 0 {
		return fmt.Errorf("YAML file %s is empty", filePath), nil
	}
	return nil, yamlFile
}

func exctractAccess(yamlBytes []byte) (error, RavelinAccess) {
	var access RavelinAccess

	if err := yaml.Unmarshal(yamlBytes, &access); err != nil {
		return fmt.Errorf("error unmarshaling IAM file: %v", err), access
	}

	return nil, access
}

func userFileToEmail(file string) string {
	return strings.ReplaceAll(strings.Split(filepath.Base(file), ".")[0], "_", ".") + "@ravelin.com"
}
