data "google_dns_managed_zone" "env_dns_zone" {
  provider  = google-beta
  name      = "gcp-jkwong-info"
  project   = data.google_project.host_project.project_id
}

resource "google_dns_record_set" "helloweb-dev" {
  provider      = google-beta
  managed_zone  = data.google_dns_managed_zone.env_dns_zone.name
  project       = data.google_project.host_project.project_id
  name          = format("helloweb-dev-%s.gcp.jkwong.info.", random_id.random_suffix.hex)
  type          = "A"
  rrdatas       = [
    google_compute_global_address.helloweb-dev.address
  ]
  ttl          = 300
}

resource "google_compute_global_address" "helloweb-dev" {
  name      = format("helloweb-dev-%s", random_id.random_suffix.hex)
  project   = module.service_project.project_id
}

resource "google_compute_global_forwarding_rule" "helloweb-dev-https" {
  name        = format("helloweb-dev-https-%s", random_id.random_suffix.hex)
  target      = google_compute_target_https_proxy.helloweb-dev.id
  port_range  = "443"
  ip_address  = google_compute_global_address.helloweb-dev.id
  load_balancing_scheme = "EXTERNAL_MANAGED"
  project     = module.service_project.project_id
}

resource "google_compute_managed_ssl_certificate" "helloweb-dev" {
  name      = format("helloweb-dev-%s", random_id.random_suffix.hex)
  project   = module.service_project.project_id

  managed {
    domains = [format("helloweb-dev-%s.gcp.jkwong.info.", random_id.random_suffix.hex)]
  }
}

resource "google_compute_target_https_proxy" "helloweb-dev" {
  name              = format("helloweb-dev-%s", random_id.random_suffix.hex)
  url_map           = google_compute_url_map.helloweb-dev.id
  ssl_certificates  = [google_compute_managed_ssl_certificate.helloweb-dev.id]
  project           = module.service_project.project_id
}

resource "google_compute_url_map" "helloweb-dev" {
  name            = format("helloweb-dev-%s", random_id.random_suffix.hex)
  description     = format("helloweb-dev-%s", random_id.random_suffix.hex)
  default_service = google_compute_backend_service.helloweb-dev-a.id
  project         = module.service_project.project_id

  host_rule {
    hosts        = [format("helloweb-dev-%s.gcp.jkwong.info", random_id.random_suffix.hex)]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = google_compute_backend_service.default.id

    route_rules {
      priority = 1000
      service = google_compute_backend_service.helloweb-dev-a.id
      match_rules {
        prefix_match = "/serviceA"
        ignore_case = true
      }
    }

    route_rules {
      priority = 1001
      service = google_compute_backend_service.helloweb-dev-b.id
      match_rules {
        prefix_match = "/serviceB"
        ignore_case = true
      }
    }
  }
}

resource "google_compute_backend_service" "helloweb-dev-b" {
  name        = format("helloweb-dev-%s-b", random_id.random_suffix.hex)
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_backend_service" "helloweb-dev-a" {
  name        = format("helloweb-dev-%s-a", random_id.random_suffix.hex)
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_backend_service" "default" {
  name        = format("helloweb-dev-default-%s", random_id.random_suffix.hex)
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 300

  load_balancing_scheme = "EXTERNAL_MANAGED"
  locality_lb_policy    = "LEAST_REQUEST"

  lifecycle {
    ignore_changes = [
      backend,
    ]
  }

  health_checks = [google_compute_health_check.http-health-check.id]
  project       = module.service_project.project_id

  log_config {
    enable = true
    sample_rate = 1
  }

}

resource "google_compute_health_check" "http-health-check" {
  name                = format("helloweb-http-health-check-%s", random_id.random_suffix.hex)
  check_interval_sec  = 3
  timeout_sec         = 1
  project             = module.service_project.project_id

  http_health_check {
    port_specification  = "USE_SERVING_PORT"
    request_path        = "/healthz"
  }

  log_config {
    enable = true
  }
}
