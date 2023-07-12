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

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestImageSyncBasic(t *testing.T) {
	srcReg := httptest.NewServer(registry.New())
	defer srcReg.Close()

	destReg := httptest.NewServer(registry.New())
	defer destReg.Close()

	fakeImg, _ := random.Image(10, 1)
	fakeImgDigest, _ := fakeImg.Digest()

	// let's pretend the tagged image source digest has changed
	fakeImgModified, _ := random.Image(10, 1)
	fakeImgDigestModified, _ := fakeImgModified.Digest()

	initSrcImage(srcReg, "library/busybox:1.0", fakeImg)
	initSrcImage(srcReg, "library/busybox:latest", fakeImg)

	stubImageSyncConfig := func(srcReg, destReg *httptest.Server, srcTag, destTag string) string {
		return fmt.Sprintf(`resource "ravelin_imagesync" "unit_test" {
			source      = "%s/library/busybox:%s"
			destination = "%s/busybox:%s"
		}`, srcReg.URL[7:], srcTag, destReg.URL[7:], destTag)
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 nil,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				// Create the resource, mirror the image to the dest registry, and correctly set the id (w/digest)
				Config:       stubImageSyncConfig(srcReg, destReg, "1.0", "1.0"),
				ResourceName: "ravelin_imagesync.unit_test",
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("ravelin_imagesync.unit_test", plancheck.ResourceActionCreate),
					},
				},
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("ravelin_imagesync.unit_test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "id", destReg.URL[7:]+"/busybox@"+fakeImgDigest.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source_digest", fakeImgDigest.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source", srcReg.URL[7:]+"/library/busybox:latest"),
				),
			},

			{
				// Test changing the digest of the source image but keep the same tag will trigger an update
				PreConfig: func() {
					initSrcImage(srcReg, "library/busybox:latest", fakeImgModified)
				},
				Config:       stubImageSyncConfig(srcReg, destReg, "latest", "1.0"),
				ResourceName: "ravelin_imagesync.unit_test",
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("ravelin_imagesync.unit_test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "id", destReg.URL[7:]+"/busybox@"+fakeImgDigestModified.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source_digest", fakeImgDigestModified.String()),
					resource.TestCheckResourceAttr("ravelin_imagesync.unit_test", "source", srcReg.URL[7:]+"/library/busybox:latest"),
				),
			},
		},
	})
}

func TestImageSyncPublicImages(t *testing.T) {

	destReg := httptest.NewServer(registry.New())
	defer destReg.Close()

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

	// Test we can pull public dockerhub images
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 nil,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
				Config:       stubImageSyncDockerhubConfig(destReg),
				ResourceName: "ravelin_imagesync.docker_unit_test",
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("ravelin_imagesync.docker_unit_test", "id"),
					resource.TestCheckResourceAttrSet("ravelin_imagesync.docker_unit_test", "source_digest"),
					resource.TestCheckResourceAttr("ravelin_imagesync.docker_unit_test", "source", "registry.hub.docker.com/library/hello-world:latest"),
				),
			},
		},
	})

	// Test we can pull public quay.io images
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		PreCheck:                 nil,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             nil,
		Steps: []resource.TestStep{
			{
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
