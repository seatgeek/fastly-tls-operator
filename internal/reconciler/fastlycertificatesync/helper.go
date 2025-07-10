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
func getCertificateAndTLSSecretFromSubject(ctx *Context) (*cmv1.Certificate, *corev1.Secret, error) {
	// get certificate from subject
	certificate := &cmv1.Certificate{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: ctx.Subject.Spec.CertificateName, Namespace: ctx.Subject.ObjectMeta.Namespace}, certificate); err != nil {
		return nil, nil, fmt.Errorf("failed to get certificate of name %s and namespace %s: %w", ctx.Subject.Spec.CertificateName, ctx.Subject.ObjectMeta.Namespace, err)
	}

	// get secret from certificate
	secret := &corev1.Secret{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: certificate.Spec.SecretName, Namespace: certificate.Namespace}, secret); err != nil {
		return nil, nil, fmt.Errorf("failed to get secret of name %s and namespace %s: %w", certificate.Spec.SecretName, certificate.Namespace, err)
	}

	return certificate, secret, nil
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

// get the certPEM byte slice for the given secret.
// abstract away the details around local reconciliation vs. trusted issuers.
func getCertPEMForSecret(ctx *Context, secret *corev1.Secret) ([]byte, error) {
	// Get certificate details from secret
	certPEM, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s does not contain tls.crt", secret.Namespace, secret.Name)
	}

	// in a local environment, we need to provide the entire chain of trust and append caCertPEM details to the certPEM
	// in a production scenario with a trusted issuer, we don't need to provide the root details since Fastly will already have them.
	if ctx.Config.HackFastlyCertificateSyncLocalReconciliation {
		ctx.Log.Info("local environment detected, appending root CA details")
		// Attempt to get the root CA certificate details from the secret, if required.
		// We cannot proceed if this is not present when in our local reconciliation mode.
		caCertPEM, ok := secret.Data["ca.crt"]
		if !ok {
			return nil, fmt.Errorf("secret %s/%s does not contain ca.crt", secret.Namespace, secret.Name)
		}
		certPEM = append(certPEM, caCertPEM...)
	}
	return certPEM, nil
}
