variable "service_project_id" {
  description = "The ID of the service project which hosts the project resources e.g. dev-55427"
}

variable "dns_project_id" {
  description = "The ID of the service project which hosts cloud dns e.g. dns-55427"
}

variable "dns_zone" {}
variable "dns_name" {}