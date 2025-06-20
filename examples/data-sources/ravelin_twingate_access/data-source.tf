data "ravelin_twingate_access" "twingate_access" {
  iam_path = "../internal/iam"
}

locals {
    twingate_access = data.ravelin_twingate_access.twingate_access.twingate_access
}

output "twingate_access" {
  value = local.twingate_access
}