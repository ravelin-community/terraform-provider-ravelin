data "ravelin_conditional_bindings" "example" {
  project = "my-project"
}

data "google_iam_policy" "project" {

  binding {
    role = "roles/editor"
    members = [
      "user:john.doe@email.com",
      "serviceAccount:service-a@my-project.iam.gserviceaccount",
    ]
  }

  // Automatically add all other conditional bindings from the data source
  dynamic "binding" {
    for_each = toset(data.ravelin_conditional_bindings.example.bindings)

    content {
      role    = binding.value.role
      members = binding.value.members
      condition {
        title       = binding.value.condition.title
        description = binding.value.condition.description
        expression  = binding.value.condition.expression
      }
    }
  }
}

resource "google_project_iam_policy" "project" {
  project     = "my-project"
  policy_data = data.google_iam_policy.project.policy_data
}
