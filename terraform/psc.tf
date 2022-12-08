module "autoneg-psc" {
    source = "./autoneg-psc"

    service_project_id = module.service_project.project_id
    dns_project_id = data.google_project.host_project.project_id
    dns_zone = "p-jkwong-info"
    dns_name = "helloweb-${random_id.random_suffix.hex}.p.jkwong.info"
    region = var.region

    shared_vpc_name = data.google_compute_network.shared_vpc.name
    host_project_id = data.google_project.host_project.project_id
    subnet = module.service_project.subnets[0].self_link

    acme_email = var.acme_email
    acme_registration = [ 
        {
            hmac_base64 = var.acme_registration_hmac_base64
            key_id = var.acme_registration_key_id
        } 
    ]

    psc_nat_subnet_cidr_range = "10.251.0.0/24"
}

output "psc_attachment_id" {
    value = module.autoneg-psc.psc_attachment_id
}

output "private_dns_name" {
    value = module.autoneg-psc.dns_name
}