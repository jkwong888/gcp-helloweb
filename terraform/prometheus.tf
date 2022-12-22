resource "google_service_account" "gmp_sa" {
  project       = module.service_project.project_id
  account_id    = "gmp-sa"
}

resource "google_service_account_iam_member" "gmp_wi" {
    depends_on = [
      google_container_cluster.primary,
    ]
    service_account_id = google_service_account.gmp_sa.id
    role = "roles/iam.workloadIdentityUser"
    member = "serviceAccount:${module.service_project.project_id}.svc.id.goog[prometheus/prometheus]"
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