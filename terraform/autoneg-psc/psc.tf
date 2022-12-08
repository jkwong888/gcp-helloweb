resource "google_compute_service_attachment" "helloweb_ilb_service_attachment" {
  project       = data.google_project.service_project.project_id
  name          = "psc-${local.app_name}-ilb"
  region        = var.region

  # observed -- proxy protocol = true results in 400 bad request errors
  enable_proxy_protocol    = false
  connection_preference    = "ACCEPT_AUTOMATIC"
  nat_subnets              = [
    google_compute_subnetwork.psc_nat_region.id
  ]
  target_service           = google_compute_forwarding_rule.helloweb_l7_ilb.id
}

resource "google_compute_subnetwork" "psc_nat_region" {
  project       = data.google_project.host_project.project_id
  name          = "psc-${local.app_name}-nat-${var.region}"
  region        = var.region

  purpose = "PRIVATE_SERVICE_CONNECT"

  ip_cidr_range = var.psc_nat_subnet_cidr_range
  network = data.google_compute_network.shared_vpc.id

}
