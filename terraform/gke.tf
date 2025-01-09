locals {
  gke_sa_roles = [
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/monitoring.viewer",
    "roles/stackdriver.resourceMetadata.writer"
  ]
} 

# resource "google_compute_firewall" "gke_master_webhook" {
#   project     = data.google_project.host_project.project_id
#   name        = format("%s-master-webhook", var.gke_cluster_name)
#   network     = data.google_compute_network.shared_vpc.id

#   allow {
#     protocol = "tcp"
#     ports = ["443", "10250"]
#   }

#   source_ranges = [var.gke_cluster_master_range]
#   target_tags = [random_id.random_network_tag.hex]
# }

# # enable GKE service to use host project networks
# resource "google_project_iam_member" "gke_host_service_agent_user" {
#   depends_on = [
#     module.service_project.enabled_apis,
#   ]

#   project     = data.google_project.host_project.project_id
#   role        = "roles/container.hostServiceAgentUser"
#   member      = format("serviceAccount:service-%d@container-engine-robot.iam.gserviceaccount.com", module.service_project.number)
# }

# # GKE node service account
# resource "google_service_account" "gke_sa" {
#   project       = module.service_project.project_id
#   account_id    = format("%s-sa", var.gke_cluster_name)
#   display_name  = format("%s cluster service account", var.gke_cluster_name)
# }

# resource "google_project_iam_member" "gke_sa_role" {
#   count     = length(local.gke_sa_roles)

#   project   = module.service_project.project_id
#   role      = element(local.gke_sa_roles, count.index)
#   member    = format("serviceAccount:%s", google_service_account.gke_sa.email)
# }


# resource "random_id" "random_network_tag" {
#   byte_length      = 2 
#   prefix           = "n"
# }

# resource "google_service_account_iam_member" "metrics_wi" {
#     depends_on = [
#       module.gke.cluster_id,
#     ]
#     service_account_id = google_service_account.metrics_sa.id
#     role = "roles/iam.workloadIdentityUser"
#     member = "serviceAccount:${module.service_project.project_id}.svc.id.goog[custom-metrics/custom-metrics-stackdriver-adapter]"
# }

# resource "google_service_account_iam_member" "helloweb_wi" {
#     depends_on = [
#       module.gke.cluster_id,
#     ]
#     for_each = google_service_account.helloweb_sa
#     service_account_id = each.value.id
#     role = "roles/iam.workloadIdentityUser"
#     member = "serviceAccount:${module.service_project.project_id}.svc.id.goog[${each.key}/${each.key}]"
# }

# resource "google_service_account_iam_member" "gmp_wi" {
#     depends_on = [
#       module.gke.cluster_id,
#     ]
#     service_account_id = google_service_account.gmp_sa.id
#     role = "roles/iam.workloadIdentityUser"
#     member = "serviceAccount:${module.service_project.project_id}.svc.id.goog[prometheus/prometheus]"
# }

# module "gke" {
#   depends_on = [
#     module.service_project.enabled_apis,
#     module.service_project.subnet_users,
#     google_project_iam_member.gke_host_service_agent_user,
#     google_project_iam_member.gke_sa_role,
#     google_project_organization_policy.shielded_vm_disable,
#     google_project_organization_policy.oslogin_disable,
#   ]

#   source = "./gke-std"
#   name = var.gke_cluster_name
#   region = var.gke_cluster_region
#   service_project_id = module.service_project.project_id
#   network = data.google_compute_network.shared_vpc.self_link
#   subnet = lookup(
#     zipmap(
#       module.service_project.subnets.*.name, 
#       module.service_project.subnets.*.self_link),
#     var.gke_cluster_subnet,
#     ""
#   )

#   pods_range_name = var.gke_cluster_subnet_pods_range_name
#   services_range_name = var.gke_cluster_subnet_services_range_name

#   private_cluster = var.gke_cluster_private_cluster     # nodes have private IPs only
#   master_cidr = var.gke_cluster_private_cluster ? var.gke_cluster_master_range : ""
 
#   service_account = google_service_account.gke_sa.email
#   network_tag = random_id.random_network_tag.hex

#   default_nodepool_initial_size = var.gke_cluster_default_nodepool_initial_size
#   default_nodepool_min_size = var.gke_cluster_default_nodepool_min_size
#   default_nodepool_max_size = var.gke_cluster_default_nodepool_max_size
#   default_nodepool_machine_type = var.gke_cluster_default_nodepool_machine_type
#   default_nodepool_use_preemptible_nodes = var.gke_cluster_use_preemptible_nodes

# }