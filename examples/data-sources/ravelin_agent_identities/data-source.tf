# Generic identity pool for all agents in the project
data "ravelin_agent_identities" "project_pool" {
  project         = "my-project"
  organization_id = "123456789012"
}

resource "google_project_iam_member" "agent_pool_binding" {
  project = "my-project"
  role    = "roles/storage.objectViewer"
  member  = data.ravelin_agent_identities.project_pool.identity
}

# Unique identity for a specific named agent
data "ravelin_agent_identities" "my_agent" {
  project         = "my-project"
  organization_id = "123456789012"
  agent_name      = "my-test-agent"
  location        = "us-central1"
}

resource "google_project_iam_member" "my_agent_binding" {
  project = "my-project"
  role    = "roles/bigquery.dataViewer"
  member  = data.ravelin_agent_identities.my_agent.identity
}
