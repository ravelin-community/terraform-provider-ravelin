data "ravelin_agent_identities" "example" {
  project = "my-project"
}


locals {
  agent_identity_policy = jsondecode(
    data.ravelin_agent_identities.example.agent_identity_policy
  )

  my_project_policy = {
    "roles/editor" = [
      "user:john.doe@email.com",
      "serviceAccount:service-a@my-project.iam.gserviceaccount",
    ],
    "roles/bigquery.admin" = [
      "user:alice.bob@email.com"
    ]
  }

  combined_policy = { for role in distinct(
    concat(
      keys(local.agent_identity_policy),
      keys(local.my_project_policy),
    )
    ) : role =>
    concat(
      lookup(local.agent_identity_policy, role, []),
      lookup(local.my_project_policy, role, []),
    )
  }
}

data "google_iam_policy" "project" {
  dynamic "binding" {
    for_each = local.combined_policy

    content {
      role    = binding.key
      members = binding.value
    }
  }
}

resource "google_project_iam_policy" "project" {
  project     = "my-project"
  policy_data = data.google_iam_policy.project.policy_data
}
