resource "ravelin_imagesync" "hello" {
  source      = "registry.hub.docker.com/library/hello-world:latest"
  destination = "europe-docker.pkg.dev/my-project/my-registry/dockerhub/hello-world:latest"
}
