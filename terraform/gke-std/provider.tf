terraform {
  required_providers {
    google = {
      version = "~> 4.44.0"
    }
    google-beta = {
      version = "~> 4.44.0"
    }
    null = {
      version = "~> 2.1"
    }
    random = {
      version = "~> 2.2"
    }
    tls = {
      source = "hashicorp/tls"
      version = "4.0.4"
    }
  }
}