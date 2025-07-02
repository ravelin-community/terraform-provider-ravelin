data "ravelin_twingate_access" "twingate_users" {
  iam_path = "../internal/iam"
}

locals {
    twingate_access = data.ravelin_twingate_access.twingate_users.twingate_access
}

output "twingate_access" {
  value = local.twingate_access
}