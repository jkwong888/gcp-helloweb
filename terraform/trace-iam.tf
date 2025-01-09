locals {
  service_accounts = toset([
    "helloweb-a",
    "helloweb-b",
  ])
}

resource "google_service_account" "helloweb_sa" {
  for_each      = local.service_accounts
  project       = module.service_project.project_id
  account_id    = each.key
}




resource "google_project_iam_member" "trace_writer" {
    for_each = google_service_account.helloweb_sa
    project = module.service_project.project_id
    role = "roles/cloudtrace.agent"
    member = "serviceAccount:${each.value.email}"
}