package iam

import (
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/ravelin-community/terraform-provider-ravelin/internal/gsudo"
	"gopkg.in/yaml.v3"
)

type IAM struct {
	Email       string
	GCPIAM      GCPAccess    `yaml:"gcp,omitempty"`   // GCP IAM roles and groups
	GsudoConfig gsudo.Config `yaml:"gsudo,omitempty"` // gsudo configuration for the user
}

type GCPAccess struct {
	Groups []string `yaml:"groups,omitempty"` // list of google workspace groups the user belongs to
}

func exctractAccess(filePath string) (error, IAM) {
	var iam IAM
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading IAM file: %v", err), iam
	}

	if err = yaml.Unmarshal(yamlFile, &iam); err != nil {
		return fmt.Errorf("error unmarshaling IAM file: %v", err), iam
	}

	return nil, iam
}

func ExtractUserAccess(iamDirectory string) (error, []IAM) {
	userAccess := make([]IAM, 0)
	err, userFiles := getUserFiles(iamDirectory)
	if err != nil {
		return fmt.Errorf("error getting user files:  %v", err), nil
	}

	for _, userFile := range userFiles {
		if !strings.HasSuffix(userFile, ".yaml") {
			log.Printf("Skipping non-YAML file: %s", userFile)
			continue
		}

		err, userIAM := exctractAccess(fmt.Sprintf("%s/users/%s", iamDirectory, userFile))
		userIAM.Email = userFileToEmail(userFile)
		if err != nil {
			log.Fatalf("error extracting user access: %v", err)
		}

		for _, group := range userIAM.GCPIAM.Groups {

			err, groupIAM := exctractAccess(fmt.Sprintf("%s/groups/%s.yaml", iamDirectory, group))
			if err != nil {
				log.Fatalf("error extracting group access for %s: %v", group, err)
			}
			maps.Copy(userIAM.GsudoConfig.Escalations, groupIAM.GsudoConfig.Escalations)
		}

		userAccess = append(userAccess, userIAM)
	}
	return nil, userAccess
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

func userFileToEmail(file string) string {
	return strings.ReplaceAll(strings.Split(filepath.Base(file), ".")[0], "_", ".") + "@ravelin.com"
}
