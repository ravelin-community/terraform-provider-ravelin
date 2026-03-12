resource "google_kms_key_ring" "attestation" {
  name     = "attestation"
  location = "europe"
}

resource "google_kms_crypto_key" "attestation" {
  name     = "image-signing"
  key_ring = google_kms_key_ring.attestation.id
  purpose  = "ASYMMETRIC_SIGN"
  version_template {
    algorithm = "EC_SIGN_P256_SHA256"
  }
}

resource "ravelin_imagesync" "hello" {
  source      = "registry.hub.docker.com/library/hello-world:latest"
  destination = "europe-docker.pkg.dev/my-project/my-registry/dockerhub/hello-world:latest"
  kms_key_id  = google_kms_crypto_key.attestation.id
}
