package provider

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ravelin-community/terraform-provider-ravelin/internal/models"
	"google.golang.org/api/cloudresourcemanager/v1"
)

func TestFilterPolicyForConditionalBindings(t *testing.T) {
	tests := []struct {
		name   string
		role   string
		input  *cloudresourcemanager.Policy
		output []*cloudresourcemanager.Binding
	}{
		{
			name: "should_return_all_conditional_bindings",
			input: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role:    "roles/servicenetworking.serviceAgent",
						Members: []string{"serviceAccount:account-a@gcp-project.iam.gserviceaccount.com"},
						Condition: &cloudresourcemanager.Expr{
							Title:      "test",
							Expression: "test",
						},
					},
					{
						Role:    "roles/editor",
						Members: []string{"serviceAccount:account-b@gcp-project.iam.gserviceaccount.com"},
						Condition: &cloudresourcemanager.Expr{
							Title:      "test",
							Expression: "test",
						},
					},
				},
			},
			output: []*cloudresourcemanager.Binding{
				{
					Role:    "roles/servicenetworking.serviceAgent",
					Members: []string{"serviceAccount:account-a@gcp-project.iam.gserviceaccount.com"},
					Condition: &cloudresourcemanager.Expr{
						Title:      "test",
						Expression: "test",
					},
				},
				{
					Role:    "roles/editor",
					Members: []string{"serviceAccount:account-b@gcp-project.iam.gserviceaccount.com"},
					Condition: &cloudresourcemanager.Expr{
						Title:      "test",
						Expression: "test",
					},
				},
			},
		},
		{
			name: "should_return_conditional_binding",
			role: "roles/editor",
			input: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{
					{
						Role:    "roles/servicenetworking.serviceAgent",
						Members: []string{"serviceAccount:account-a@gcp-project.iam.gserviceaccount.com"},
						Condition: &cloudresourcemanager.Expr{
							Title:      "test",
							Expression: "test",
						},
					},
					{
						Role:    "roles/editor",
						Members: []string{"serviceAccount:account-a@gcp-project.iam.gserviceaccount.com"},
						Condition: &cloudresourcemanager.Expr{
							Title:      "test",
							Expression: "test",
						},
					},
				},
			},
			output: []*cloudresourcemanager.Binding{
				{
					Role:    "roles/editor",
					Members: []string{"serviceAccount:account-a@gcp-project.iam.gserviceaccount.com"},
					Condition: &cloudresourcemanager.Expr{
						Title:      "test",
						Expression: "test",
					},
				},
			},
		},
		{
			name: "should_return_empty_array",
			role: "roles/editor",
			input: &cloudresourcemanager.Policy{
				Bindings: []*cloudresourcemanager.Binding{},
			},
			output: []*cloudresourcemanager.Binding{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := filterPolicyForConditionalBindings(tt.input, tt.role)
			if diff := cmp.Diff(tt.output, got); diff != "" {
				t.Fatalf("Expected (-) but got (+):\n%s", diff)
			}
		})
	}
}

func TestNewConditionalBinding(t *testing.T) {
	tests := []struct {
		name     string
		input    *cloudresourcemanager.Binding
		expected models.ConditionalBindingModel
		wantErr  bool
	}{
		{
			name: "valid binding",
			input: &cloudresourcemanager.Binding{
				Role: "roles/viewer",
				Members: []string{
					"user:test@example.com",
					"serviceAccount:test@project.iam.gserviceaccount.com",
				},
				Condition: &cloudresourcemanager.Expr{
					Title:       "test-title",
					Description: "test-description",
					Expression:  "test-expression",
				},
			},
			expected: models.ConditionalBindingModel{
				Role: types.StringValue("roles/viewer"),
				Members: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("user:test@example.com"),
					types.StringValue("serviceAccount:test@project.iam.gserviceaccount.com"),
				}),
				Condition: types.ObjectValueMust(
					map[string]attr.Type{
						"title":       types.StringType,
						"description": types.StringType,
						"expression":  types.StringType,
					},
					map[string]attr.Value{
						"title":       types.StringValue("test-title"),
						"description": types.StringValue("test-description"),
						"expression":  types.StringValue("test-expression"),
					},
				),
			},
			wantErr: false,
		},
		{
			name: "empty binding",
			input: &cloudresourcemanager.Binding{
				Role:      "",
				Members:   []string{},
				Condition: &cloudresourcemanager.Expr{},
			},
			expected: models.ConditionalBindingModel{
				Role:    types.StringValue(""),
				Members: types.ListValueMust(types.StringType, []attr.Value{}),
				Condition: types.ObjectValueMust(
					map[string]attr.Type{
						"title":       types.StringType,
						"description": types.StringType,
						"expression":  types.StringType,
					},
					map[string]attr.Value{
						"title":       types.StringValue(""),
						"description": types.StringValue(""),
						"expression":  types.StringValue(""),
					},
				),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := newConditionalBinding(context.Background(), tt.input)

			if tt.wantErr {
				if !diags.HasError() {
					t.Errorf("newConditionalBinding() expected error but got none")
				}
				return
			}

			if diags.HasError() {
				t.Errorf("newConditionalBinding() unexpected error: %v", diags)
				return
			}

			// Compare role
			if got.Role != tt.expected.Role {
				t.Errorf("newConditionalBinding() role = %v, want %v", got.Role, tt.expected.Role)
			}

			// Compare members
			if !got.Members.Equal(tt.expected.Members) {
				t.Errorf("newConditionalBinding() members = %v, want %v", got.Members, tt.expected.Members)
			}

			// Compare condition
			if !got.Condition.Equal(tt.expected.Condition) {
				t.Errorf("newConditionalBinding() condition = %v, want %v", got.Condition, tt.expected.Condition)
			}
		})
	}
}
