variable "shared_vpc_name" {
  description = "name of the shared vpc"
}

variable "host_project_id" {
  description = "The ID of the host project which hosts the vpc e.g. vpc-55427"
}

variable "service_project_id" {
  description = "The ID of the service project which hosts the project resources e.g. dev-55427"
}

variable "dns_project_id" {
  description = "The ID of the service project which hosts cloud dns e.g. dns-55427"
}

variable "dns_zone" {}
variable "dns_name" {}

variable "region" {}
variable "subnet" {}

variable acme_email {}
variable acme_registration {
  description = "some acme providers require external account bindings (e.g. google publicca)"
  type = list(object({
    key_id = string
    hmac_base64 = string
  }))
}

variable "psc_nat_subnet_cidr_range" {}