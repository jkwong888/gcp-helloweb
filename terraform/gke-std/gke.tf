resource "google_container_cluster" "primary" {
  provider = google-beta

  lifecycle {
    ignore_changes = [
      # Ignore changes to tags, e.g. because a management agent
      # updates these based on some ruleset managed elsewhere.
      node_config,
    ]
  }

  name     = var.name
  location = var.region
  project  = var.service_project_id

  release_channel  {
    channel = "REGULAR"
  }

  # We can't create a cluster with no node pool defined, but we want to only use
  # separately managed node pools. So we create the smallest possible default
  # node pool and immediately delete it.
  remove_default_node_pool = true
  initial_node_count       = 1

  private_cluster_config {
    enable_private_nodes = var.private_cluster     # nodes have private IPs only
    enable_private_endpoint = false  # master nodes private IP only
    master_ipv4_cidr_block = var.master_cidr
  }

  master_authorized_networks_config {
    cidr_blocks {
      cidr_block = "0.0.0.0/0"
      display_name = "eerbody"
    }
  }

  network = var.network
  subnetwork = var.subnet

  vertical_pod_autoscaling {
    enabled = true
  }
  datapath_provider = "ADVANCED_DATAPATH"

  networking_mode = "VPC_NATIVE"
  ip_allocation_policy {
    cluster_secondary_range_name = var.pods_range_name
    services_secondary_range_name = var.services_range_name
  }

  workload_identity_config {
    workload_pool = "${var.service_project_id}.svc.id.goog"
  }

  cluster_autoscaling {
    enabled = false # this settings is for nodepool autoprovisioning
    autoscaling_profile = "OPTIMIZE_UTILIZATION"
  }

  dns_config {
    cluster_dns = "CLOUD_DNS"
    cluster_dns_scope = "CLUSTER_SCOPE"
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

  name       = format("%s-default-pvm", var.name)
  location   = var.region
  cluster    = var.name
  node_count = var.default_nodepool_initial_size
  project    = var.service_project_id

  autoscaling {
    min_node_count = var.default_nodepool_min_size
    max_node_count = var.default_nodepool_max_size
  }

  node_config {
    spot = var.default_nodepool_use_preemptible_nodes
    machine_type = var.default_nodepool_machine_type

    metadata = {
      disable-legacy-endpoints = "true"
    }

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    service_account = var.service_account
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]

    tags = [
      var.network_tag
    ]
  }
}

