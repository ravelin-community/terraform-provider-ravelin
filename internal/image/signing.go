package image

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intotov1 "github.com/in-toto/attestation/go/v1"
	cbundle "github.com/sigstore/cosign/v3/pkg/cosign/bundle"
	ociremote "github.com/sigstore/cosign/v3/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/v3/pkg/signature"
	"github.com/sigstore/cosign/v3/pkg/types"
	sigsig "github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/dsse"
	signatureoptions "github.com/sigstore/sigstore/pkg/signature/options"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

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
	digestParts := strings.Split(digestRef.DigestStr(), ":")
	if len(digestParts) != 2 {
		return fmt.Errorf("unable to parse digest %s", digestRef.DigestStr())
	}
	statement := &intotov1.Statement{
		Type: intotov1.StatementTypeUri,
		Subject: []*intotov1.ResourceDescriptor{{
			Digest: map[string]string{digestParts[0]: digestParts[1]},
		}},
		PredicateType: types.CosignSignPredicateType,
		Predicate:     &structpb.Struct{},
	}
	payload, err := protojson.Marshal(statement)
	if err != nil {
		return fmt.Errorf("marshal statement: %w", err)
	}

	// DSSE-sign the statement. The private key never leaves GCP KMS.
	wrappedSigner := dsse.WrapSigner(sv, types.IntotoPayloadType)
	signedPayload, err := wrappedSigner.SignMessage(bytes.NewReader(payload), signatureoptions.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("DSSE sign: %w", err)
	}

	pubKey, err := sv.PublicKey(signatureoptions.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("get public key: %w", err)
	}
	signerPEM, err := sigs.PublicKeyPem(sv, signatureoptions.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("get public key PEM: %w", err)
	}

	// Assemble a Sigstore protobuf bundle (v0.3) from the DSSE envelope.
	bundleBytes, err := cbundle.MakeNewBundle(pubKey, nil, payload, signedPayload, signerPEM, nil)
	if err != nil {
		return fmt.Errorf("create bundle: %w", err)
	}

	remoteOpt := ociremote.WithRemoteOptions(remote.WithAuth(auth))
	if err := ociremote.WriteAttestationNewBundleFormat(digestRef, bundleBytes, types.CosignSignPredicateType, remoteOpt); err != nil {
		return fmt.Errorf("push bundle: %w", err)
	}

	return nil
}
