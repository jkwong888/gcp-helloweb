module "autoneg-glb" {
    source = "./autoneg-glb"

    service_project_id = module.service_project.project_id
    dns_project_id = data.google_project.host_project.project_id
    dns_zone = "gcp-jkwong-info"
    dns_name = "helloweb-${random_id.random_suffix.hex}.gcp.jkwong.info"
}

output "public_dns_name" {
    value = module.autoneg-glb.dns_name
}

