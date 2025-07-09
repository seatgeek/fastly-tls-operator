package fastlycertificatesync

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fastly/go-fastly/v10/fastly"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultFastlyPageSize = 20
)

func (l *Logic) fastlyPrivateKeyExists(ctx *Context) (bool, error) {

	// get certificate from subject
	certificate := &cmv1.Certificate{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: ctx.Subject.Spec.CertificateName, Namespace: ctx.Subject.ObjectMeta.Namespace}, certificate); err != nil {
		return false, fmt.Errorf("failed to get certificate of name %s and namespace %s: %w", ctx.Subject.Spec.CertificateName, ctx.Subject.ObjectMeta.Namespace, err)
	}

	// get secret from certificate
	secret := &corev1.Secret{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: certificate.Spec.SecretName, Namespace: certificate.Namespace}, secret); err != nil {
		return false, fmt.Errorf("failed to get secret of name %s and namespace %s: %w", certificate.Spec.SecretName, certificate.Namespace, err)
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

// GetPublicKeySHA1FromPEM calculates the SHA1 hash of the public key derived from a PEM-encoded private key
func getPublicKeySHA1FromPEM(keyPEM []byte) (string, error) {

	// Decode the PEM block
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block")
	}

	// Parse the private key as an RSA key
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	// Extract the public key (it is part of the RSA private key)
	pubKey := &priv.PublicKey

	// Marshal the public key to DER format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}

	// Encode the public key to PEM format
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// Compute the SHA1 hash of the PEM-encoded public key
	hash := sha1.New()
	hash.Write(pubKeyPEM)
	hashValue := hash.Sum(nil)

	sha1String := hex.EncodeToString(hashValue)
	return sha1String, nil
}
