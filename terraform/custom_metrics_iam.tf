resource "google_service_account" "metrics_sa" {
  project       = module.service_project.project_id
  account_id    = "custom-metrics"
}


resource "google_service_account_iam_member" "metrics_wi" {
    service_account_id = google_service_account.metrics_sa.id
    role = "roles/iam.workloadIdentityUser"
    member = "serviceAccount:${module.service_project.project_id}.svc.id.goog[custom-metrics/custom-metrics-stackdriver-adapter]"
}

resource "google_project_iam_member" "metrics_viewer" {
    project = module.service_project.project_id
    role = "roles/monitoring.viewer"
    member = "serviceAccount:${google_service_account.metrics_sa.email}"
}