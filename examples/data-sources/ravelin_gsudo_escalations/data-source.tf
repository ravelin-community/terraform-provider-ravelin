data "ravelin_gsudo_escalations" "gsudo_escalations" {
  iam_path = "../internal/iam"
}

locals {
  gsudo_escalations = data.ravelin_gsudo_escalations.gsudo_escalations
}

output "gsudo_escalations" {
  value = local.gsudo_escalations.escalations
}