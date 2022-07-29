module "autoneg-gcp" {
  source = "git@github.com:GoogleCloudPlatform/gke-autoneg-controller//terraform/gcp"

  shared_vpc = {
    project_id                = data.google_project.host_project.project_id
    subnetwork_region         = var.region
    subnetwork_id             = google_container_cluster.primary.subnetwork
  }
  project_id                  = module.service_project.project_id

  workload_identity = {
      namespace = "autoneg-system"
      service_account = "autoneg"
  }
}