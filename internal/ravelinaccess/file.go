package ravelinaccess

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readFile reads a file and returns its content, it returns an error if the file is empty.
func readFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filePath, err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file %s is empty", filePath)
	}
	return data, nil
}

// GetUserFiles returns a list of user files from the IAM directory
func GetUserFiles(iamDirectory string) ([]string, error) {
	files, err := os.ReadDir(filepath.Join(iamDirectory, "users"))
	if err != nil {
		return nil, fmt.Errorf("error retrieving a list of user files from IAM directory: %v", err)
	}

	var userFiles []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yml") || strings.HasSuffix(file.Name(), ".yaml")) {
			userFiles = append(userFiles, file.Name())
		}
	}

	return userFiles, nil
}

// fileToType returns the type of entity based on the path of the yaml file
func fileToType(file string) (EntityType, error) {
	switch {
	case filepath.Dir(file) == "users" || strings.HasSuffix(filepath.Dir(file), "/users"):
		return USER, nil
	case filepath.Dir(file) == "service-accounts" || strings.HasSuffix(filepath.Dir(file), "/service-accounts"):
		return SERVICE, nil
	case filepath.Dir(file) == "groups" || strings.HasSuffix(filepath.Dir(file), "/groups"):
		return GROUP, nil
	}

	return -1, errors.New("unable to determine type of file")
}

// fileToEmail returns the email of the entity based on the path of the yaml file
func fileToEmail(file string, typ EntityType) (string, error) {
	switch typ {
	case USER:
		parts := strings.Split(filepath.Base(file), ".")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid user file: %s, expected format: <name>_<surname>.yml", file)
		}
		return strings.ReplaceAll(parts[0], "_", ".") + "@ravelin.com", nil

	case GROUP:
		parts := strings.Split(filepath.Base(file), ".")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid group file: %s, expected format: <group-name>.yml", file)
		}
		return "gcp-" + parts[0] + "@ravelin.com", nil

	case SERVICE:
		return "", errors.New("service accounts are not yet supported")
	}

	return "", fmt.Errorf("invalid entity type for file: %s", file)
}
