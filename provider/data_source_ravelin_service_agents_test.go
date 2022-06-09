package provider

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"google.golang.org/api/cloudresourcemanager/v1"
)

func TestFilterPolicy(t *testing.T) {
	tests := []struct {
		name          string
		inputPolicy   *cloudresourcemanager.Policy
		projectNumber string
		outputPolicy  map[string][]string
	}{
		{
			name: "filter_simple_policy",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/servicenetworking.serviceAgent",
						Members: []string{
							"serviceAccount:service-239645365406@service-networking.iam.gserviceaccount.com",
						},
					},
				},
			},
			projectNumber: "239645365406",
			outputPolicy: map[string][]string{
				"roles/servicenetworking.serviceAgent": {
					"serviceAccount:service-239645365406@service-networking.iam.gserviceaccount.com",
				},
			},
		},
		{
			name: "check_project_number",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/servicenetworking.serviceAgent",
						Members: []string{
							"serviceAccount:service-239645365406@service-networking.iam.gserviceaccount.com",
						},
					},
					{
						Role: "roles/editor",
						Members: []string{
							"serviceAccount:service-239645365406@containerregistry.iam.gserviceaccount.com",
						},
					},
				},
			},
			projectNumber: "239645365407",
			outputPolicy:  map[string][]string{},
		},
		{
			name: "check_regex_domain",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role: "roles/servicenetworking.serviceAgent",
						Members: []string{
							"serviceAccount:service-239645365406@gcp-sa-should-be-service-agent.iam.gserviceaccount.com",
						},
					},
					{
						Role: "roles/editor",
						Members: []string{
							"serviceAccount:service-239645365406@gcpsa-should-be-service-agent.iam.gserviceaccount.com",
						},
					},
				},
			},
			projectNumber: "239645365406",
			outputPolicy: map[string][]string{
				"roles/servicenetworking.serviceAgent": {
					"serviceAccount:service-239645365406@gcp-sa-should-be-service-agent.iam.gserviceaccount.com",
				},
			},
		},
		{
			name: "check cloudbuild is not included",
			inputPolicy: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role:    "roles/editor",
						Members: []string{"serviceAccount:239645365406@cloudbuild.gserviceaccount.com"},
					},
				},
			},
			projectNumber: "239645365406",
			outputPolicy:  map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := filterPolicy(tt.inputPolicy, tt.projectNumber)
			if diff := cmp.Diff(tt.outputPolicy, got); diff != "" {
				t.Fatalf("Expected stats (-) but got (+):\n%s", diff)
			}
		})
	}
}
