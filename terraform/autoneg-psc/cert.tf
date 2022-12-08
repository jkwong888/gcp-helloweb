resource "tls_private_key" "private_key" {
  algorithm = "RSA"
}

resource "acme_registration" "reg" {
  depends_on = [
    google_project_service.publicca_project_api,
  ]

  account_key_pem = tls_private_key.private_key.private_key_pem
  email_address   = var.acme_email

  dynamic "external_account_binding" {
    for_each = var.acme_registration
    content {
        key_id = external_account_binding.value["key_id"]
        hmac_base64 = external_account_binding.value["hmac_base64"]
    }
  }

}

resource "acme_certificate" "certificate" {
  depends_on = [
    google_project_service.publicca_project_api,
  ]

  account_key_pem           = acme_registration.reg.account_key_pem
  common_name               = var.dns_name

  dns_challenge {
    provider = "gcloud"
    config = {
      "GCE_PROJECT" = data.google_project.dns_project.project_id
    }
  }
}