module "autoneg-gcp" {
  depends_on = [
    module.gke.cluster_id,
  ]
  source = "git@github.com:GoogleCloudPlatform/gke-autoneg-controller//terraform/gcp"

  shared_vpc = {
    project_id                = data.google_project.host_project.project_id
    subnetwork_region         = var.region
    subnetwork_id             = module.gke.subnetwork_id
  }
  project_id                  = module.service_project.project_id

  workload_identity = {
      namespace = "autoneg-system"
      service_account = "autoneg"
  }
}