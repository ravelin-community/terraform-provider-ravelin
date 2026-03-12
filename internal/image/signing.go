package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/v3/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/v3/pkg/oci/remote"
	"github.com/sigstore/cosign/v3/pkg/oci/static"
	sigs "github.com/sigstore/cosign/v3/pkg/signature"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	signatureoptions "github.com/sigstore/sigstore/pkg/signature/options"
	sigPayload "github.com/sigstore/sigstore/pkg/signature/payload"

	// Register the GCP KMS provider so gcpkms:// URIs are recognised.
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
)

// SignImage loads the GCP KMS signer and signs the image.
func SignImage(ctx context.Context, digestRef name.Digest, kmsRef string, auth authn.Authenticator) error {
	sv, err := sigs.SignerVerifierFromKeyRef(ctx, "gcpkms://"+kmsRef, nil, nil)
	if err != nil {
		return fmt.Errorf("load KMS signer: %w", err)
	}
	return signImage(ctx, digestRef, sv, auth)
}

// signImage is the testable core: it signs digestRef using the provided signer
// and pushes the OCI signature to the registry via the referrers API.
func signImage(ctx context.Context, digestRef name.Digest, sv sigsig.Signer, auth authn.Authenticator) error {
	payload, err := (&sigPayload.Cosign{Image: digestRef}).MarshalJSON()
	if err != nil {
		return fmt.Errorf("build payload: %w", err)
	}

	rawSig, err := sv.SignMessage(bytes.NewReader(payload), signatureoptions.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	b64Sig := base64.StdEncoding.EncodeToString(rawSig)

	ociSig, err := static.NewSignature(payload, b64Sig)
	if err != nil {
		return fmt.Errorf("create OCI signature: %w", err)
	}

	remoteOpt := ociremote.WithRemoteOptions(remote.WithAuth(auth))
	se := ociremote.SignedUnknown(digestRef, remoteOpt)

	newSE, err := mutate.AttachSignatureToEntity(se, ociSig)
	if err != nil {
		return fmt.Errorf("attach signature: %w", err)
	}

	if err := ociremote.WriteSignaturesExperimentalOCI(digestRef, newSE, remoteOpt); err != nil {
		return fmt.Errorf("push signature: %w", err)
	}

	return nil
}