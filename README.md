# terraform-provider-ravelin

Miscellaneous resources and operations we couldn't do natively in terraform.

## ravelin_service_agents Data Resource

`ravelin_service_agents` data resource is used to dynamically fetch all the service agents and their roles in the project IAM policy. This is particularly useful when trying to use authoritative policies in GCP as service agents can appear/dissapear depending on the APIs enabled in your project. 

### Example Usage

```hcl
terraform {
  required_providers {
    ravelin = {
      source  = "ravelin-community/ravelin"
      version = "1.0.0"
    }
  }
}

provider "ravelin" {}

data "ravelin_service_agents" "test" {
  project = "google_project123"
}

locals {
  service_agent_policy = jsondecode(data.ravelin_service_agents.test.service_agent_policy)
}

output "example" {
  value = local.service_agent_policy
}

```

The output would something like:

```sh
Changes to Outputs:
  + example = {
      + roles/cloudbuild.serviceAgent        = [
          + "serviceAccount:service-239645365406@gcp-sa-cloudbuild.iam.gserviceaccount.com",
        ]
      + roles/compute.serviceAgent           = [
          + "serviceAccount:service-239645365406@compute-system.iam.gserviceaccount.com",
        ]
      + roles/container.serviceAgent         = [
          + "serviceAccount:service-239645365406@container-engine-robot.iam.gserviceaccount.com",
        ]
      + roles/editor                         = [
          + "serviceAccount:service-239645365406@containerregistry.iam.gserviceaccount.com",
        ]
      + roles/file.serviceAgent              = [
          + "serviceAccount:service-239645365406@cloud-filer.iam.gserviceaccount.com",
        ]
      + roles/ml.serviceAgent                = [
          + "serviceAccount:service-239645365406@cloud-ml.google.com.iam.gserviceaccount.com",
        ]
      + roles/servicenetworking.serviceAgent = [
          + "serviceAccount:service-239645365406@service-networking.iam.gserviceaccount.com",
        ]
    }
```

### Usage Notes

**Reference projects by string ID not by project number**

All GCP projects both have a project ID string (that you choose when creating the project) and a randomly assigned 12 digit project number. Please use the `ravelin_service_agents` data resource with your project ID string.

**Service agents across different projects**

The data resource will only return service agents intended to be used with your current project. All other service agents from different projects won't be added to the output if they are part of the project IAM policy.