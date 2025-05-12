package google

import (
	"context"
	"strconv"

	"google.golang.org/api/cloudresourcemanager/v1"
)

type Config struct {
	CloudResourceManagerService *cloudresourcemanager.Service
}

func (c *Config) NewCloudResourceManagerService(ctx context.Context) error {
	service, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return err
	}
	c.CloudResourceManagerService = service
	return nil
}

func (c *Config) GetProjectIAMPolicy(projectID string) (*cloudresourcemanager.Policy, error) {
	request := cloudresourcemanager.GetIamPolicyRequest{
		Options: &cloudresourcemanager.GetPolicyOptions{
			RequestedPolicyVersion: 3,
		},
	}
	policy, err := c.CloudResourceManagerService.Projects.GetIamPolicy(projectID, &request).Do()

	if err != nil {
		return nil, err
	}
	return policy, err
}

func (c *Config) GetProjectIDNumber(projectID string) (string, error) {
	project, err := c.CloudResourceManagerService.Projects.Get(projectID).Do()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(project.ProjectNumber, 10), nil
}
