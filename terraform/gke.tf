locals {
  gke_sa_roles = [
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/monitoring.viewer",
    "roles/stackdriver.resourceMetadata.writer"
  ]
} 

resource "google_compute_firewall" "gke_master_webhook" {
  project     = data.google_project.host_project.project_id
  name        = format("%s-master-webhook", var.gke_cluster_name)
  network     = data.google_compute_network.shared_vpc.id

  allow {
    protocol = "tcp"
    ports = ["443", "10250"]
  }

  source_ranges = [var.gke_cluster_master_range]
  target_tags = [random_string.random_network_tag.result]
}

# enable GKE service to use host project networks
resource "google_project_iam_member" "gke_host_service_agent_user" {
  depends_on = [
    module.service_project.enabled_apis,
  ]

  project     = data.google_project.host_project.project_id
  role        = "roles/container.hostServiceAgentUser"
  member      = format("serviceAccount:service-%d@container-engine-robot.iam.gserviceaccount.com", module.service_project.number)
}

# GKE node service account
resource "google_service_account" "gke_sa" {
  project       = module.service_project.project_id
  account_id    = format("%s-sa", var.gke_cluster_name)
  display_name  = format("%s cluster service account", var.gke_cluster_name)
}

resource "google_project_iam_member" "gke_sa_role" {
  count     = length(local.gke_sa_roles)

  project   = module.service_project.project_id
  role      = element(local.gke_sa_roles, count.index)
  member    = format("serviceAccount:%s", google_service_account.gke_sa.email)
}

resource "google_container_cluster" "primary" {
  provider = google-beta

  lifecycle {
    ignore_changes = [
      # Ignore changes to tags, e.g. because a management agent
      # updates these based on some ruleset managed elsewhere.
      node_config,
    ]
  }

  depends_on = [
    module.service_project.enabled_apis,
    module.service_project.subnet_users,
    google_project_iam_member.gke_host_service_agent_user,
    google_project_iam_member.gke_sa_role,
    google_project_organization_policy.shielded_vm_disable,
    google_project_organization_policy.oslogin_disable,
  ]

  name     = var.gke_cluster_name
  location = var.gke_cluster_region
  project  = module.service_project.project_id

  release_channel  {
    channel = "REGULAR"
  }

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1

  private_cluster_config {
    enable_private_nodes = var.gke_cluster_private_cluster     # nodes have private IPs only
    enable_private_endpoint = false  # master nodes private IP only
    master_ipv4_cidr_block = var.gke_cluster_private_cluster ? var.gke_cluster_master_range : ""
  }

  master_authorized_networks_config {
    cidr_blocks {
      cidr_block = "0.0.0.0/0"
      display_name = "eerbody"
    }
  }

  network = data.google_compute_network.shared_vpc.self_link
  subnetwork = lookup(
    zipmap(
      module.service_project.subnets.*.name, 
      module.service_project.subnets.*.self_link),
    var.gke_cluster_subnet,
    ""
  )

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {
    cluster_secondary_range_name = var.gke_cluster_subnet_pods_range_name
    services_secondary_range_name = var.gke_cluster_subnet_services_range_name
  }

  workload_identity_config {
    workload_pool = "${module.service_project.project_id}.svc.id.goog"
  }

  cluster_autoscaling {
    enabled = false # this settings is for nodepool autoprovisioning
    autoscaling_profile = "OPTIMIZE_UTILIZATION"
  }

}

resource "google_container_node_pool" "primary_preemptible_nodes" {
  lifecycle {
    ignore_changes = [
      node_count,
    ]
  }

  depends_on = [
    google_container_cluster.primary,
  ]

  name       = format("%s-default-pvm", var.gke_cluster_name)
  location   = var.gke_cluster_region
  cluster    = var.gke_cluster_name
  node_count = var.gke_cluster_default_nodepool_initial_size
  project    = module.service_project.project_id

  autoscaling {
    min_node_count = var.gke_cluster_default_nodepool_min_size
    max_node_count = var.gke_cluster_default_nodepool_max_size
  }

  node_config {
    preemptible  = var.gke_cluster_use_preemptible_nodes
    machine_type = var.gke_cluster_default_nodepool_machine_type

    metadata = {
      disable-legacy-endpoints = "true"
    }

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    service_account = google_service_account.gke_sa.email
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    tags = [
      random_string.random_network_tag.result
    ]
  }
}

resource "random_string" "random_network_tag" {
  length           = 10
  special          = true
  override_special = "-"
  upper            = false
}