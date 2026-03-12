package image

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"github.com/stretchr/testify/require"
)

// pushRandomImage pushes a random image to the registry at addr and returns its
// digest reference.
func pushRandomImage(t *testing.T, addr string) name.Digest {
	t.Helper()

	img, err := random.Image(512, 1)
	require.NoError(t, err)

	ref, err := name.ParseReference(addr+"/test/hello:latest", name.Insecure)
	require.NoError(t, err)

	require.NoError(t, remote.Write(ref, img, remote.WithAuth(authn.Anonymous)))

	digest, err := img.Digest()
	require.NoError(t, err)

	digestRef, err := name.NewDigest(addr+"/test/hello@"+digest.String(), name.Insecure)
	require.NoError(t, err)

	return digestRef
}

// ecdsaSigner returns a sigstore Signer backed by a freshly generated P-256 key.
func ecdsaSigner(t *testing.T) sigsig.Signer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	sv, err := sigsig.LoadECDSASignerVerifier(key, crypto.SHA256)
	require.NoError(t, err)
	return sv
}

func TestSignImage_Success(t *testing.T) {
	srv := httptest.NewServer(registry.New())
	defer srv.Close()

	addr := strings.TrimPrefix(srv.URL, "http://")
	digestRef := pushRandomImage(t, addr)

	err := signImage(context.Background(), digestRef, ecdsaSigner(t), authn.Anonymous)
	require.NoError(t, err)
}
