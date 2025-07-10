package fastlycertificatesync

import (
	"fmt"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fastly/go-fastly/v10/fastly"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultFastlyPageSize = 20
)

func (l *Logic) getFastlyPrivateKeyExists(ctx *Context) (bool, error) {

	secret, err := getTLSSecretFromSubject(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get TLS secret from context: %w", err)
	}

	// get private key from secret
	keyPEM, ok := secret.Data["tls.key"]
	if !ok {
		return false, fmt.Errorf("secret %s/%s does not contain tls.key", secret.Namespace, secret.Name)
	}

	var allPrivateKeys []*fastly.PrivateKey
	pageNumber := 1

	for {
		privateKeys, err := l.FastlyClient.ListPrivateKeys(&fastly.ListPrivateKeysInput{
			PageNumber: pageNumber,
			PageSize:   defaultFastlyPageSize,
		})
		if err != nil {
			return false, fmt.Errorf("failed to list Fastly private keys: %w", err)
		}

		allPrivateKeys = append(allPrivateKeys, privateKeys...)

		// If we received fewer keys than the page size, we've reached the end
		if len(privateKeys) < defaultFastlyPageSize {
			break
		}
		pageNumber++
	}

	// Fastly doesn't advertise the private key values from its API (this is good)
	// They will instead give us the sha1 of the public key component, which we can calculate on our end in order to match against the private key.
	publicKeySHA1, err := getPublicKeySHA1FromPEM(keyPEM)
	if err != nil {
		return false, fmt.Errorf("failed to get public key SHA1: %w", err)
	}

	ctx.Log.Info("calculated public key SHA1", "sha1", publicKeySHA1)

	// does a private key exist in Fastly with a matching public key sha1?
	keyExistsInFastly := false
	for _, key := range allPrivateKeys {
		ctx.Log.V(5).Info("found private key in Fastly with public_key_sha1", "public_key_sha1", key.PublicKeySHA1)
		if key.PublicKeySHA1 == publicKeySHA1 {
			ctx.Log.Info("found matching private key in Fastly, we do not need to upload our key", "key_id", key.ID, "fastly_public_key_sha1", key.PublicKeySHA1, "local_public_key_sha1", publicKeySHA1)
			keyExistsInFastly = true
		}
	}

	return keyExistsInFastly, nil
}

func (l *Logic) createFastlyPrivateKey(ctx *Context) error {

	secret, err := getTLSSecretFromSubject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get TLS secret from context: %w", err)
	}

	keyPEM, ok := secret.Data["tls.key"]
	if !ok {
		return fmt.Errorf("secret %s/%s does not contain tls.key", secret.Namespace, secret.Name)
	}

	createResp, err := l.FastlyClient.CreatePrivateKey(&fastly.CreatePrivateKeyInput{
		Key:  string(keyPEM),
		Name: secret.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to create Fastly private key: %w", err)
	}
	ctx.Log.Info("created new private key in Fastly", "key_id", createResp.ID)

	return nil
}

func (l *Logic) getFastlyCertificateStatus(ctx *Context) (CertificateStatus, error) {

	fastlyCertificate, err := l.getFastlyCertificateMatchingSubject(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Fastly certificate matching subject: %w", err)
	}

	// an empty certificate means the certificate is not present in Fastly
	if fastlyCertificate == nil {
		return CertificateStatusMissing, nil
	}
	// is the returned certificate up to date with the subject certificate?
	return CertificateStatusStale, nil

	//return "", nil
}

// Get the Fastly certificate whose details match the certificate referenced by the subject
func (l *Logic) getFastlyCertificateMatchingSubject(ctx *Context) (*fastly.CustomTLSCertificate, error) {

	subjectCertificate := &cmv1.Certificate{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: ctx.Subject.Spec.CertificateName, Namespace: ctx.Subject.ObjectMeta.Namespace}, subjectCertificate); err != nil {
		return nil, fmt.Errorf("failed to get certificate of name %s and namespace %s: %w", ctx.Subject.Spec.CertificateName, ctx.Subject.ObjectMeta.Namespace, err)
	}

	// List existing certificates in Fastly
	var allCerts []*fastly.CustomTLSCertificate
	pageNumber := 1

	for {
		certs, err := l.FastlyClient.ListCustomTLSCertificates(&fastly.ListCustomTLSCertificatesInput{
			PageNumber: pageNumber,
			PageSize:   defaultFastlyPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list Fastly certificates: %w", err)
		}

		allCerts = append(allCerts, certs...)

		// If we received fewer certificates than the page size, we've reached the end
		if len(certs) < defaultFastlyPageSize {
			break
		}
		pageNumber++
	}

	ctx.Log.Info(fmt.Sprintf("found %d certificates", len(allCerts)))

	// match certificate based on name
	for _, cert := range allCerts {
		if cert.Name == subjectCertificate.Name {
			return cert, nil
		}
	}

	// no match found
	return nil, nil
}
