resource "google_service_account" "gmp_sa" {
  project       = module.service_project.project_id
  account_id    = "gmp-sa"
}



resource "google_project_iam_member" "gmp_metrics_viewer" {
    project = module.service_project.project_id
    role = "roles/monitoring.viewer"
    member = "serviceAccount:${google_service_account.gmp_sa.email}"
}

resource "google_project_iam_member" "gmp_metrics_writer" {
    project = module.service_project.project_id
    role = "roles/monitoring.metricWriter"
    member = "serviceAccount:${google_service_account.gmp_sa.email}"
}