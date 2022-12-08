output "dns_name" {
    value = var.dns_name
}

output "psc_attachment_id" {
    value = google_compute_service_attachment.helloweb_ilb_service_attachment.id
}
