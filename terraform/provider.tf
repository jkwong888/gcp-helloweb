terraform {
  backend "gcs" {
    bucket  = "jkwng-altostrat-com-tf-state"
    prefix = "jkwng-helloweb-dev"
  }

  required_providers {
    google = {
      version = "~> 5.0"
    }
    google-beta = {
      version = "~> 5.0"

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

    acme = {
      source = "vancluever/acme"
      version = "2.11.1"
    }

  }
}

provider "google" {
#  credentials = file(local.credentials_file_path)
}

provider "google-beta" {
#  credentials = file(local.credentials_file_path)
}

provider "null" {
}

provider "random" {
}

provider "acme" {
  server_url = "https://dv.acme-v02.api.pki.goog/directory"
}
