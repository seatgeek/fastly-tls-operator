package fastlycertificatesync

import (
	"fmt"

	"github.com/fastly/go-fastly/v10/fastly"
)

const (
	defaultFastlyPageSize = 20
)

func (l *Logic) getFastlyPrivateKeyExists(ctx *Context) (bool, error) {

	secret, err := getTLSSecretFromContext(ctx)
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

	secret, err := getTLSSecretFromContext(ctx)
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

func (l *Logic) getFastlyCertificateStatus(ctx *Context) (*CertificateStatus, error) {
	return nil, nil
}
