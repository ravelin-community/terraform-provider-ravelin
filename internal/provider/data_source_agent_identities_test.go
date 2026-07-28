package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"google.golang.org/api/cloudresourcemanager/v1"
)

func TestFilterAgentIdentities(t *testing.T) {
	tests := []struct {
		name          string
		inputPolicy   *cloudresourcemanager.Policy
		projectNumber string
		role          string
		outputPolicy  map[string][]string
	}{
		{
			name: "filter_agent_identity",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "",
			outputPolicy: map[string][]string{
				"roles/aiplatform.user": {
					"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
				},
			},
		},
		{
			name: "filter_out_different_project",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
				},
			},
			projectNumber: "999999999999",
			role:          "",
			outputPolicy:  map[string][]string{},
		},
		{
			name: "filter_out_service_accounts",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
							"serviceAccount:service-308055776777@gcp-sa-aiplatform.iam.gserviceaccount.com",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "",
			outputPolicy: map[string][]string{
				"roles/aiplatform.user": {
					"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
				},
			},
		},
		{
			name: "filter_out_conditional_bindings",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
						Condition: &cloudresourcemanager.Expr{
							Expression: "request.time < timestamp('2024-01-01T00:00:00Z')",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "",
			outputPolicy:  map[string][]string{},
		},
		{
			name: "multiple_roles",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
					{
						Role: "roles/storage.objectViewer",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "",
			outputPolicy: map[string][]string{
				"roles/aiplatform.user": {
					"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
				},
				"roles/storage.objectViewer": {
					"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
				},
			},
		},
		{
			name: "different_org_numbers",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-111111111111.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
							"principalSet://agents.global.org-222222222222.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "",
			outputPolicy: map[string][]string{
				"roles/aiplatform.user": {
					"principalSet://agents.global.org-111111111111.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
					"principalSet://agents.global.org-222222222222.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
				},
			},
		},
		{
			name: "filter_by_role",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
					{
						Role: "roles/storage.objectViewer",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "roles/aiplatform.user",
			outputPolicy: map[string][]string{
				"roles/aiplatform.user": {
					"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
				},
			},
		},
		{
			name: "filter_by_role_no_match",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/aiplatform.user",
						Members: []string{
							"principalSet://agents.global.org-743276511950.system.id.goog/attribute.platformContainer/aiplatform/projects/308055776777",
						},
					},
				},
			},
			projectNumber: "308055776777",
			role:          "roles/storage.objectViewer",
			outputPolicy:  map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := filterAgentIdentities(tt.inputPolicy, tt.projectNumber, tt.role)
			if diff := cmp.Diff(tt.outputPolicy, got); diff != "" {
				t.Fatalf("Expected output (-) but got (+):\n%s", diff)
			}
		})
	}
}
