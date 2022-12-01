data "google_dns_managed_zone" "env_dns_zone" {
  provider  = google-beta
  name      = "gcp-jkwong-info"
  project   = data.google_project.host_project.project_id
}

resource "google_dns_record_set" "helloweb-dev" {
  provider      = google-beta
  managed_zone  = data.google_dns_managed_zone.env_dns_zone.name
  project       = data.google_project.host_project.project_id
  name          = "helloweb-${random_id.random_suffix.hex}.gcp.jkwong.info."
  type          = "A"
  rrdatas       = [
    google_compute_global_address.helloweb-dev.address
  ]
  ttl          = 300
}

resource "google_compute_global_address" "helloweb-dev" {
  name      = "helloweb-${random_id.random_suffix.hex}"
  project   = module.service_project.project_id
}

resource "google_compute_global_forwarding_rule" "helloweb-dev-https" {
  name        = "helloweb-dev-https-${random_id.random_suffix.hex}"
  target      = google_compute_target_https_proxy.helloweb-dev.id
  port_range  = "443"
  ip_address  = google_compute_global_address.helloweb-dev.id
  load_balancing_scheme = "EXTERNAL_MANAGED"
  project     = module.service_project.project_id
}

resource "google_compute_global_forwarding_rule" "helloweb-dev-http" {
  project     = module.service_project.project_id

  name        = "helloweb-dev-http-${random_id.random_suffix.hex}"
  target      = google_compute_target_http_proxy.helloweb-dev-http.id
  port_range  = "80"
  ip_address  = google_compute_global_address.helloweb-dev.id
  load_balancing_scheme = "EXTERNAL_MANAGED"
}

resource "google_compute_target_http_proxy" "helloweb-dev-http" {
  project           = module.service_project.project_id
  name              = format("helloweb-dev-http-%s", random_id.random_suffix.hex)
  url_map           = google_compute_url_map.helloweb-dev-http.id
}

resource "google_compute_url_map" "helloweb-dev-http" {
  name            = format("helloweb-dev-http-%s", random_id.random_suffix.hex)
  description     = format("helloweb-dev--http-%s", random_id.random_suffix.hex)
  project         = module.service_project.project_id

  default_url_redirect {
    https_redirect = true
    strip_query = false
  }
}

resource "google_compute_target_https_proxy" "helloweb-dev" {
  name              = format("helloweb-dev-%s", random_id.random_suffix.hex)
  url_map           = google_compute_url_map.helloweb-dev.id
  project           = module.service_project.project_id
  certificate_map   = "//certificatemanager.googleapis.com/${google_certificate_manager_certificate_map.helloweb.id}"
}

resource "google_compute_url_map" "helloweb-dev" {
  name            = format("helloweb-dev-%s", random_id.random_suffix.hex)
  description     = format("helloweb-dev-%s", random_id.random_suffix.hex)
  default_service = google_compute_backend_service.helloweb-dev-a.id
  project         = module.service_project.project_id

  host_rule {
    hosts        = [format("helloweb-%s.gcp.jkwong.info", random_id.random_suffix.hex)]
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

resource "google_compute_global_address" "helloweb_ip_address" {
  project   = module.service_project.project_id
  name      = "helloweb-glb-${random_id.random_suffix.hex}"
}

resource "google_certificate_manager_certificate_map" "helloweb" {
  project   = module.service_project.project_id
  name      = "helloweb-cert-map-${random_id.random_suffix.hex}"
}

resource "google_certificate_manager_certificate_map_entry" "helloweb-default" {
  project   = module.service_project.project_id
  name      = "helloweb-cert-map-default-${random_id.random_suffix.hex}"

  map = google_certificate_manager_certificate_map.helloweb.name 
  certificates = [google_certificate_manager_certificate.cert.id]

  matcher = "PRIMARY"
}

resource "google_certificate_manager_certificate" "cert" {
  project   = module.service_project.project_id
  name = "helloweb-${random_id.random_suffix.hex}"
  scope = "DEFAULT"
  managed {
    domains = [
      //"helloweb-${random_id.random_suffix.hex}.gcp.jkwong.info",
      "*.gcp.jkwong.info",
    ]

    dns_authorizations = [
      //google_certificate_manager_dns_authorization.helloweb.id,
      google_certificate_manager_dns_authorization.wildcard.id,
    ]
  }
}

resource "google_certificate_manager_dns_authorization" "helloweb" {
  project     = module.service_project.project_id
  name        = "helloweb-${random_id.random_suffix.hex}-dns-auth"
  domain      = "helloweb-${random_id.random_suffix.hex}.gcp.jkwong.info"
}

resource "google_certificate_manager_dns_authorization" "wildcard" {
  project     = module.service_project.project_id
  name        = "wildcard-dns-auth-${random_id.random_suffix.hex}"
  domain      = "gcp.jkwong.info"
}

resource "google_dns_record_set" "helloweb_auth" {
  depends_on = [
    google_certificate_manager_dns_authorization.helloweb,
  ]

  project   = data.google_project.host_project.project_id

  name = google_certificate_manager_dns_authorization.helloweb.dns_resource_record.0.name
  type = google_certificate_manager_dns_authorization.helloweb.dns_resource_record.0.type
  ttl  = 5

  managed_zone = data.google_dns_managed_zone.env_dns_zone.name

  rrdatas = [
    google_certificate_manager_dns_authorization.helloweb.dns_resource_record.0.data

  ]
}

resource "google_dns_record_set" "wildcard_auth" {
  depends_on = [
    google_certificate_manager_dns_authorization.wildcard,
  ]

  project   = data.google_project.host_project.project_id

  name = google_certificate_manager_dns_authorization.wildcard.dns_resource_record.0.name
  type = google_certificate_manager_dns_authorization.wildcard.dns_resource_record.0.type
  ttl  = 5

  managed_zone = data.google_dns_managed_zone.env_dns_zone.name

  rrdatas = [
    google_certificate_manager_dns_authorization.wildcard.dns_resource_record.0.data

  ]
}
