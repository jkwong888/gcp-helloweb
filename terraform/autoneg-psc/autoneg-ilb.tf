
data "google_compute_network" "shared_vpc" {
  project = data.google_project.host_project.project_id
  name = var.shared_vpc_name
}

data "google_dns_managed_zone" "env_dns_zone" {
  provider  = google-beta
  name      = var.dns_zone
  project   = data.google_project.dns_project.project_id
}

locals {
  app_name = split(".", var.dns_name)[0]
  dns_domain_split = split(".", var.dns_name)
  dns_domain = join(".", slice(local.dns_domain_split, 1, length(local.dns_domain_split)))
}

resource "google_dns_record_set" "helloweb-dev" {
  provider      = google-beta
  managed_zone  = data.google_dns_managed_zone.env_dns_zone.name
  project       = data.google_project.dns_project.project_id
  name          = "${var.dns_name}."
  type          = "A"
  rrdatas       = [
    google_compute_address.helloweb_ip_address.address
  ]
  ttl          = 300
}

resource "google_compute_address" "helloweb_ip_address" {
  project   = data.google_project.service_project.project_id
  name      = "${local.app_name}-ilb-address-${random_id.random_suffix.hex}"

  address_type = "INTERNAL"
  region       = var.region
  subnetwork   = var.subnet
}

resource "google_compute_region_ssl_certificate" "helloweb" {
  project     = data.google_project.service_project.project_id
  region      = var.region
  name        = replace(var.dns_name, ".", "-")
  certificate = "${acme_certificate.certificate.certificate_pem}${acme_certificate.certificate.issuer_pem}" 
  private_key = acme_certificate.certificate.private_key_pem
}

resource "google_compute_region_target_https_proxy" "helloweb" {
  project  = data.google_project.service_project.project_id
  name     = "${local.app_name}-https-proxy-${random_id.random_suffix.hex}"
  provider = google-beta
  region   = var.region
  url_map  = google_compute_region_url_map.helloweb-dev.id
  ssl_certificates = [
    google_compute_region_ssl_certificate.helloweb.id,
  ]
}

resource "google_compute_forwarding_rule" "helloweb_l7_ilb" {
  project               = data.google_project.service_project.project_id
  name                  = "${local.app_name}-l7-ilb-${random_id.random_suffix.hex}"
  region                = var.region
  ip_protocol           = "TCP"
  load_balancing_scheme = "INTERNAL_MANAGED"
  port_range            = 443
  allow_global_access   = true
  target                = google_compute_region_target_https_proxy.helloweb.id
  network               = data.google_compute_network.shared_vpc.id
  subnetwork            = var.subnet
  ip_address            = google_compute_address.helloweb_ip_address.self_link
  network_tier          = "PREMIUM"
}

resource "google_compute_region_url_map" "helloweb-dev" {
  name            = "${local.app_name}-ilb-${random_id.random_suffix.hex}"
  description     = "${local.app_name}-ilb-${random_id.random_suffix.hex}"
  default_service = google_compute_region_backend_service.helloweb-dev-a.id
  project         = data.google_project.service_project.project_id
  region          = var.region

  host_rule {
    hosts        = ["*"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_region_backend_service.default.id

    route_rules {
      priority = 1000
      service = google_compute_region_backend_service.helloweb-dev-a.id
      match_rules {
        prefix_match = "/serviceA"
        ignore_case = true
      }
    }

    route_rules {
      priority = 1001
      service = google_compute_region_backend_service.helloweb-dev-b.id
      match_rules {
        prefix_match = "/serviceB"
        ignore_case = true
      }
    }
  }
}

resource "google_compute_region_backend_service" "helloweb-dev-b" {
  name        = "${local.app_name}-${random_id.random_suffix.hex}-b"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 300
  region      = var.region

  load_balancing_scheme = "INTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = data.google_project.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_region_backend_service" "helloweb-dev-a" {
  name        = "${local.app_name}-${random_id.random_suffix.hex}-a"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 300
  region      = var.region

  load_balancing_scheme = "INTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = data.google_project.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_region_backend_service" "default" {
  name        = "${local.app_name}-default-${random_id.random_suffix.hex}"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 300
  region      = var.region

  load_balancing_scheme = "INTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = data.google_project.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_health_check" "http-health-check" {
  name                = "${local.app_name}-http-health-check-${random_id.random_suffix.hex}"
  check_interval_sec  = 3
  timeout_sec         = 1
  project             = data.google_project.service_project.project_id

  http_health_check {
    port_specification  = "USE_SERVING_PORT"
    request_path        = "/healthz"
  }

  log_config {
    enable = true
  }
}

