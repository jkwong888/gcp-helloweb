resource "google_service_account" "metrics_sa" {
  project       = module.service_project.project_id
  account_id    = "custom-metrics"
}


resource "google_project_iam_member" "metrics_viewer" {
    project = module.service_project.project_id
    role = "roles/monitoring.viewer"
    member = "serviceAccount:${google_service_account.metrics_sa.email}"
}