terraform {
  required_providers {
    ravelin = {
      source  = "ravelin-community/ravelin"
      version = ">= 1.0.0"
    }
  }
}

provider "ravelin" {
  project = "my-project"
}