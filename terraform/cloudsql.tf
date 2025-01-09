resource "google_sql_database_instance" "main" {
  project          = module.service_project.project_id  
  name             = "hellodb"
  database_version = "POSTGRES_15"
  region           = var.region

  settings {
    # Second-generation instance tiers are based on the machine
    # type. See argument reference below.
    tier = "db-f1-micro"

    ip_configuration {
        psc_config {
            psc_enabled = true
            allowed_consumer_projects = [module.service_project.project_id]
        }
        ipv4_enabled = false
    }

    disk_autoresize = true
    disk_type = "PD_HDD"
  }

  
}