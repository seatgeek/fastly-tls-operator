package fastlycertificatesync

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Helper function to retrieve the TLS secret from the context.
// Gets the certificate from the subject reference, and then gets the secret from the certificate reference.
func getTLSSecretFromContext(ctx *Context) (*corev1.Secret, error) {
	// get certificate from subject
	certificate := &cmv1.Certificate{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: ctx.Subject.Spec.CertificateName, Namespace: ctx.Subject.ObjectMeta.Namespace}, certificate); err != nil {
		return nil, fmt.Errorf("failed to get certificate of name %s and namespace %s: %w", ctx.Subject.Spec.CertificateName, ctx.Subject.ObjectMeta.Namespace, err)
	}

	// get secret from certificate
	secret := &corev1.Secret{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: certificate.Spec.SecretName, Namespace: certificate.Namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret of name %s and namespace %s: %w", certificate.Spec.SecretName, certificate.Namespace, err)
	}

	return secret, nil
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
