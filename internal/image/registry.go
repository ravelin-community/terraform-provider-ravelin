package image

import (
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// getRemoteImage returns a remote image if it exists along with a string
// representation of it's digest
func GetRemoteImage(url string, auth authn.Authenticator) (v1.Image, bool, string, error) {
	urlRef, err := name.ParseReference(url, name.WeakValidation)
	if err != nil {
		return empty.Image, false, "", err
	}

	i, err := remote.Image(urlRef, remote.WithAuth(auth))
	if err != nil {
		if tErr, ok := (err).(*transport.Error); ok && tErr.StatusCode == 404 {
			return empty.Image, false, "", nil
		}
		return empty.Image, false, "", err
	}

	imgDigest, err := i.Digest()
	if err != nil {
		return empty.Image, true, "", err
	}

	return i, true, imgDigest.String(), nil
}

// imageID is the fully qualified URL to the image, with any tags replaced with
// the sha256 digest instead
func ImageID(url string, img v1.Image) (string, error) {
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

// digestFromReference strips all content preceding the digest for the given,
// fully qualified, reference. If no digest is present, the resulting string
// will be empty
func DigestFromReference(ref string) string {
	at := strings.LastIndex(ref, "@")
	if at == -1 {
		return ""
	}

	if at+1 >= len(ref) {
		return ""
	}

	return ref[at+1:]
}
