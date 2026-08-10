package google

import (
	"context"
	"fmt"

	"google.golang.org/api/aiplatform/v1"
	"google.golang.org/api/option"
)

func (c *Config) NewAiplatformService(ctx context.Context, location string) error {
	endpoint := fmt.Sprintf("https://%s-aiplatform.googleapis.com/", location)
	service, err := aiplatform.NewService(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		return err
	}
	c.AiplatformService = service
	return nil
}

// GetReasoningEngineByDisplayName finds a reasoning engine by display name, returning the engine
// resource. Errors if zero or multiple engines match.
func (c *Config) GetReasoningEngineByDisplayName(ctx context.Context, projectID, location, displayName string) (*aiplatform.GoogleCloudAiplatformV1ReasoningEngine, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	filter := fmt.Sprintf(`display_name="%s"`, displayName)

	resp, err := c.AiplatformService.Projects.Locations.ReasoningEngines.List(parent).Filter(filter).PageSize(2).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing reasoning engines: %w", err)
	}

	switch len(resp.ReasoningEngines) {
	case 0:
		return nil, fmt.Errorf("no reasoning engine found with display name %q in %s", displayName, parent)
	case 1:
		return resp.ReasoningEngines[0], nil
	default:
		return nil, fmt.Errorf("multiple reasoning engines found with display name %q in %s", displayName, parent)
	}
}
