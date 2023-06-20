package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func imagesync() *schema.Resource {
	return &schema.Resource{
		Create: imagesyncCreate,
		Update: imagesyncUpdate,
		Read:   imagesyncRead,
		Delete: imagesyncDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"source": {
				Type:     schema.TypeString,
				Required: true,
			},
			"destination": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source_digest": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
		},

		CustomizeDiff: sourceChangedDiffFunc,
	}
}

func imagesyncCreate(d *schema.ResourceData, m interface{}) error {
	src := d.Get("source").(string)
	srcImg, exists, err := getRemoteImage(src)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("unable to locate source image at '%s'", src)
	}

	dest := d.Get("destination").(string)
	destRef, err := name.ParseReference(dest, name.WeakValidation)
	if err != nil {
		return err
	}

	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		return err
	}
	authOpt := remote.WithAuth(googleAuth)

	if err := remote.Write(destRef, srcImg, authOpt); err != nil {
		return err
	}

	return imagesyncRead(d, m) // Resync the state to ensure the digest and ID of state match remote img
}

func imagesyncUpdate(d *schema.ResourceData, m interface{}) error {
	// Updates can only be triggered by 'source' changes that *don't* change the 'source_digest', suggesting a
	// new registry/tag, but not a new underlying image. No actual update is necessary.
	return imagesyncRead(d, m)
}

func imagesyncRead(d *schema.ResourceData, meta interface{}) error {
	dest := d.Get("destination").(string)
	destImg, exists, err := getRemoteImage(dest)
	if err != nil {
		return err
	}

	if !exists {
		d.SetId("")
	} else {
		imgID, err := imageID(dest, destImg)
		if err != nil {
			return err
		}
		d.SetId(imgID)
	}

	return nil
}

func imagesyncDelete(d *schema.ResourceData, m interface{}) error {
	dest := d.Get("destination").(string)
	destRef, err := name.ParseReference(dest, name.WeakValidation)
	if err != nil {
		return err
	}

	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		return err
	}
	authOpt := remote.WithAuth(googleAuth)

	// Delete this tag. Perform this regardless of if other tags exist
	if err := remote.Delete(destRef, authOpt); err != nil {
		return err
	}

	// Check through all available tags to see if there are any more images referencing these blobs
	tags, err := remote.List(destRef.Context(), authOpt)
	if err != nil {
		if strings.Contains(err.Error(), "METHOD_UNKNOWN") {
			// If the registry doesn't support listing images, we can't be sure we can safely delete these blobs
			return nil
		}
		return err
	}

	for _, t := range tags {
		imgRef, err := name.ParseReference(destRef.Context().String()+":"+t, name.WeakValidation)
		if err != nil {
			return err
		}

		i, err := remote.Image(imgRef, authOpt)
		if err != nil {
			return err
		}

		imageID, err := i.Digest()
		if err != nil {
			return err
		}

		if imageID.String() == digestFromReference(d.Id()) {
			return nil // Another image is using the same layers as we are, do not delete these layers!
		}
	}

	// No other tag references these layers, we're free to delete
	idRef, err := name.ParseReference(d.Id(), name.WeakValidation)
	if err != nil {
		return err
	}

	return remote.Delete(idRef, authOpt)
}

func sourceChangedDiffFunc(ctx context.Context, d *schema.ResourceDiff, v interface{}) error {
	// Several things could have changed with the 'source', it could be that:
	// - the user wants to use a different image
	// - the source image in the registry has changed
	// - the user wants the same image, but from a different registry
	// If the first 2 are true, the digest will change, and so 'ForceNew' will be triggered,
	// If the image digest remains the same, then the resource will not be marked for update
	src := d.Get("source").(string)
	srcImg, exists, err := getRemoteImage(src)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("unable to locate source image at '%s'", src)
	}
	srcDigest, err := srcImg.Digest()
	if err != nil {
		return err
	}

	oldDigest := d.Get("source_digest").(string)
	newDigest := srcDigest.String()
	if oldDigest != newDigest {
		d.SetNew("source_digest", newDigest)
	}

	return nil
}

func getRemoteImage(url string) (v1.Image, bool, error) {
	urlRef, err := name.ParseReference(url, name.WeakValidation)
	if err != nil {
		return empty.Image, false, err
	}

	googleAuth, err := google.NewEnvAuthenticator()
	if err != nil {
		return empty.Image, false, err
	}
	authOpt := remote.WithAuth(googleAuth)

	i, err := remote.Image(urlRef, authOpt)
	if err != nil {
		if tErr, ok := (err).(*transport.Error); ok && tErr.StatusCode == 404 {
			return empty.Image, false, nil
		}
		return empty.Image, false, err
	}

	return i, true, nil
}

// imageID is the fully qualified URL to the image, with any tags replaced with the sha256 digest instead
func imageID(url string, img v1.Image) (string, error) {
	if hasSHA, _ := regexp.MatchString("(.+)(@sha256:)([a-f0-9]{64})", url); hasSHA {
		return url, nil
	}

	// Trim any tags from the url
	trimTo := strings.LastIndex(url, ":")
	if trimTo != -1 && trimTo < len(url) {
		url = url[:trimTo]
	}

	digest, err := img.Digest()
	if err != nil {
		return "", err
	}

	return url + "@" + digest.String(), nil
}

// digestFromReference strips all content preceding the digest for the given, fully qualified, reference. If
// no digest is present, the resulting string will be empty
func digestFromReference(ref string) string {
	at := strings.LastIndex(ref, "@")
	if at == -1 {
		return ""
	}

	if at+1 >= len(ref) {
		return ""
	}

	return ref[at+1:]
}
