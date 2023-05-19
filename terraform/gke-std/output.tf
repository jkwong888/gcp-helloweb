output "cluster_id" {
    value = google_container_cluster.primary.id
}

output "subnetwork_id" {
    value = google_container_cluster.primary.subnetwork
}