package provider

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry" // Modified to allow registry deletes
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestImageSync(t *testing.T) {
	srcReg := httptest.NewServer(registry.New())
	defer srcReg.Close()

	destReg := httptest.NewServer(registry.New())
	defer destReg.Close()

	fakeImg, _ := random.Image(10, 1)
	fakeImgDigest, _ := fakeImg.Digest()
	initSrcImage(srcReg, "library/busybox:1.0", fakeImg)
	initSrcImage(srcReg, "library/busybox:latest", fakeImg)

	stubImageSyncConfig := func(srcReg, destReg *httptest.Server, srcTag, destTag string) string {
		a := fmt.Sprintf(`
	resource "ravelin_imagesync" "unit_test" {
		source      = "%s/library/busybox:%s"
		destination = "%s/busybox:%s"
	}`, srcReg.URL[7:], srcTag, destReg.URL[7:], destTag)

		return a
	}

	stubImageSyncDockerhubConfig := func(destReg *httptest.Server) string {
		return fmt.Sprintf(`resource "ravelin_imagesync" "docker_unit_test" {
			source      = "registry.hub.docker.com/library/hello-world:latest"
			destination = "%s/hello-world:latest"
		}`, destReg.URL[7:])
	}

	stubImageSyncQuayConfig := func(destReg *httptest.Server) string {
		return fmt.Sprintf(`resource "ravelin_imagesync" "quay_unit_test" {
			source      = "quay.io/podman/hello:latest"
			destination = "%s/quay.io/podman/hello:latest"
		}`, destReg.URL[7:])
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:   true,
		PreCheck:     nil,
		Providers:    map[string]*schema.Provider{"ravelin": Provider()},
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			{
				// Create the resource, mirror the image to the dest registry, and correctly set the id (w/digest)
				Config:       stubImageSyncConfig(srcReg, destReg, "1.0", "1.0"),
				ResourceName: "ravelin_imagesync.unit_test",
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "id", destReg.URL[7:]+"/busybox@"+fakeImgDigest.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source_digest", fakeImgDigest.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source", srcReg.URL[7:]+"/library/busybox:1.0"),
				),
			},

			{
				// Test that updating the source, but with the same digest, does not trigger an update
				Config:       stubImageSyncConfig(srcReg, destReg, "latest", "1.0"),
				ResourceName: "ravelin_imagesync.unit_test",
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "id", destReg.URL[7:]+"/busybox@"+fakeImgDigest.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source_digest", fakeImgDigest.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source", srcReg.URL[7:]+"/library/busybox:latest"),
				),
			},

			{
				// Test we can pull public dockerhub images
				Config:       stubImageSyncDockerhubConfig(destReg),
				ResourceName: "ravelin_imagesync.docker_unit_test",
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ravelin_imagesync.docker_unit_test", "id"),
					resource.TestCheckResourceAttrSet("ravelin_imagesync.docker_unit_test", "source_digest"),
					resource.TestCheckResourceAttr("ravelin_imagesync.docker_unit_test", "source", "registry.hub.docker.com/library/hello-world:latest"),
				),
			},

			{
				// Test we can pull public quay.io images
				Config:       stubImageSyncQuayConfig(destReg),
				ResourceName: "ravelin_imagesync.quay_unit_test",
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ravelin_imagesync.quay_unit_test", "id"),
					resource.TestCheckResourceAttrSet("ravelin_imagesync.quay_unit_test", "source_digest"),
					resource.TestCheckResourceAttr("ravelin_imagesync.quay_unit_test", "source", "quay.io/podman/hello:latest"),
				),
			},
		},
	})
}

func initSrcImage(fakeReg *httptest.Server, path string, img v1.Image) {
	ref, err := name.ParseReference(fakeReg.URL[7:]+"/"+path, name.WeakValidation)
	if err != nil {
		panic(err)
	}

	if err := remote.Write(ref, img); err != nil {
		panic(err)
	}
}
