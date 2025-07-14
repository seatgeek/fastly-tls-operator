package fastlycertificatesync

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/fastly/go-fastly/v10/fastly"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultFastlyPageSize = 20
)

// joinErrors combines multiple errors into a single error
func joinErrors(errs []error) error {
	return errors.Join(errs...)
}

func (l *Logic) getFastlyPrivateKeyExists(ctx *Context) (bool, error) {

	_, secret, err := getCertificateAndTLSSecretFromSubject(ctx)
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

	_, secret, err := getCertificateAndTLSSecretFromSubject(ctx)
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

	// Empty fastlyCertificates means the certificate is not present in Fastly and must be created
	if fastlyCertificate == nil {
		return CertificateStatusMissing, nil
	}

	isFastlyCertificateStale, err := l.isFastlyCertificateStale(ctx, fastlyCertificate)
	if err != nil {
		return "", fmt.Errorf("failed to check if certificate is stale: %w", err)
	}

	// Stale fastlyCertificates will be updated with the latest local certificate
	if isFastlyCertificateStale {
		return CertificateStatusStale, nil
	}

	// Non-stale fastlyCertificates are in sync with the local certificate and do not need to be updated
	return CertificateStatusSynced, nil
}

// Get the Fastly certificate whose details match the certificate referenced by the subject
func (l *Logic) getFastlyCertificateMatchingSubject(ctx *Context) (*fastly.CustomTLSCertificate, error) {

	subjectCertificate := &cmv1.Certificate{}
	if err := ctx.Client.Client.Get(ctx, types.NamespacedName{Name: ctx.Subject.Spec.CertificateName, Namespace: ctx.Subject.Namespace}, subjectCertificate); err != nil {
		return nil, fmt.Errorf("failed to get certificate of name %s and namespace %s: %w", ctx.Subject.Spec.CertificateName, ctx.Subject.Namespace, err)
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

func (l *Logic) createFastlyCertificate(ctx *Context) error {

	subjectCertificate, tlsSecret, err := getCertificateAndTLSSecretFromSubject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get TLS secret from context: %w", err)
	}

	certPEM, err := getCertPEMForSecret(ctx, tlsSecret)
	if err != nil {
		return fmt.Errorf("failed to get CertPEM for Fastly certificate: %w", err)
	}

	_, err = l.FastlyClient.CreateCustomTLSCertificate(&fastly.CreateCustomTLSCertificateInput{
		CertBlob:           string(certPEM),
		Name:               subjectCertificate.Name,
		AllowUntrustedRoot: ctx.Config.HackFastlyCertificateSyncLocalReconciliation,
	})
	if err != nil {
		return fmt.Errorf("failed to create Fastly certificate: %w", err)
	}

	return nil
}

func (l *Logic) updateFastlyCertificate(ctx *Context) error {
	subjectCertificate, tlsSecret, err := getCertificateAndTLSSecretFromSubject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get TLS secret from context: %w", err)
	}

	certPEM, err := getCertPEMForSecret(ctx, tlsSecret)
	if err != nil {
		return fmt.Errorf("failed to get CertPEM for Fastly certificate: %w", err)
	}

	fastlyCertificate, err := l.getFastlyCertificateMatchingSubject(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Fastly certificate matching subject: %w", err)
	}

	if fastlyCertificate == nil {
		return fmt.Errorf("fastly certificate not found")
	}

	_, err = l.FastlyClient.UpdateCustomTLSCertificate(&fastly.UpdateCustomTLSCertificateInput{
		CertBlob:           string(certPEM),
		Name:               subjectCertificate.Name,
		ID:                 fastlyCertificate.ID,
		AllowUntrustedRoot: ctx.Config.HackFastlyCertificateSyncLocalReconciliation,
	})
	if err != nil {
		return fmt.Errorf("failed to update Fastly certificate: %w", err)
	}

	return nil
}

func (l *Logic) isFastlyCertificateStale(ctx *Context, fastlyCertificate *fastly.CustomTLSCertificate) (bool, error) {

	subjectCertificate, tlsSecret, err := getCertificateAndTLSSecretFromSubject(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get TLS secret from context: %w", err)
	}

	certPEM, err := getCertPEMForSecret(ctx, tlsSecret)
	if err != nil {
		return false, fmt.Errorf("failed to get cert PEM for secret: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false, fmt.Errorf("failed to decode PEM block")
	}

	// serialNumber comparison is used to determine if the local certificate was refreshed
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate: %w", err)
	}
	serialNumber := cert.SerialNumber.String()

	ctx.Log.Info("checking serial number of existing fastly certificate against local value", "domains", subjectCertificate.Spec.DNSNames, "fastly_cert_serial_number", fastlyCertificate.SerialNumber, "local_cert_serial_number", serialNumber)

	// Differing serial numbers indicates that the fastlyCertificate doesn't match local and is stale
	isStale := fastlyCertificate.SerialNumber != serialNumber
	return isStale, nil
}

func (l *Logic) getFastlyTLSActivationState(ctx *Context) ([]TLSActivationData, []string, error) {

	missingTLSActivationData := []TLSActivationData{}
	extraTLSActivationIDs := []string{}

	fastlyCertificate, err := l.getFastlyCertificateMatchingSubject(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Fastly certificate matching subject: %w", err)
	}

	domainAndConfigurationToActivation, err := l.getFastlyDomainAndConfigurationToActivationMap(ctx, fastlyCertificate)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get Fastly domain and configuration to activation map: %w", err)
	}

	// For each certificate domain and expected configuration id, report activations that do not exist
	for _, domain := range fastlyCertificate.Domains {
		for _, configID := range ctx.Subject.Spec.TLSConfigurationIds {
			if _, exists := domainAndConfigurationToActivation[domain.ID][configID]; !exists {
				missingTLSActivationData = append(missingTLSActivationData, TLSActivationData{
					Certificate:   fastlyCertificate,
					Configuration: &fastly.TLSConfiguration{ID: configID},
					Domain:        domain,
				})
			} else {
				ctx.Log.Info("TLS activation already exists", "config_id", configID)
				// Remove from map since we want to keep this activation
				delete(domainAndConfigurationToActivation[domain.ID], configID)
			}
		}
	}

	// Any remaining activations in the map should be deleted
	for _, configToActivation := range domainAndConfigurationToActivation {
		for _, activation := range configToActivation {
			extraTLSActivationIDs = append(extraTLSActivationIDs, activation.ID)
		}
	}

	return missingTLSActivationData, extraTLSActivationIDs, nil
}

// Build the mapping of domain -> configuration -> activation for a given certificate
func (l *Logic) getFastlyDomainAndConfigurationToActivationMap(ctx *Context, cert *fastly.CustomTLSCertificate) (map[string]map[string]*fastly.TLSActivation, error) {
	var allActivations []*fastly.TLSActivation
	pageNumber := 1

	for {
		activations, err := l.FastlyClient.ListTLSActivations(&fastly.ListTLSActivationsInput{
			FilterTLSCertificateID: cert.ID,
			PageNumber:             pageNumber,
			PageSize:               defaultFastlyPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list Fastly TLS activations: %w", err)
		}

		allActivations = append(allActivations, activations...)

		// If we received fewer activations than the page size, we've reached the end
		if len(activations) < defaultFastlyPageSize {
			break
		}
		pageNumber++
	}

	ctx.Log.Info(fmt.Sprintf("Found %d TLS activations", len(allActivations)), "domains", cert.Domains)

	// map domain id -> configuration id -> activation
	domainAndConfigurationToActivation := make(map[string]map[string]*fastly.TLSActivation)
	for _, activation := range allActivations {
		if domainAndConfigurationToActivation[activation.Domain.ID] == nil {
			domainAndConfigurationToActivation[activation.Domain.ID] = make(map[string]*fastly.TLSActivation)
		}
		domainAndConfigurationToActivation[activation.Domain.ID][activation.Configuration.ID] = activation
	}
	return domainAndConfigurationToActivation, nil
}

func (l *Logic) createMissingFastlyTLSActivations(_ *Context) error {
	var errors []error

	for _, activationData := range l.ObservedState.MissingTLSActivationData {
		// Create new activation
		_, err := l.FastlyClient.CreateTLSActivation(&fastly.CreateTLSActivationInput{
			Certificate:   activationData.Certificate,
			Configuration: activationData.Configuration,
			Domain:        activationData.Domain,
		})
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to create TLS activation for config %s: %w", activationData.Configuration.ID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to create TLS activations: %w", joinErrors(errors))
	}
	return nil
}

func (l *Logic) deleteExtraFastlyTLSActivations(_ *Context) error {
	var errors []error

	for _, activationID := range l.ObservedState.ExtraTLSActivationIDs {
		err := l.FastlyClient.DeleteTLSActivation(&fastly.DeleteTLSActivationInput{ID: activationID})
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to delete TLS activation %s: %w", activationID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to delete TLS activations: %w", joinErrors(errors))
	}
	return nil
}

func (l *Logic) getFastlyUnusedPrivateKeyIDs(_ *Context) ([]string, error) {
	privateKeys, err := l.FastlyClient.ListPrivateKeys(&fastly.ListPrivateKeysInput{FilterInUse: "false"})
	if err != nil {
		return nil, fmt.Errorf("failed to list Fastly private keys: %w", err)
	}

	unusedPrivateKeyIDs := []string{}
	for _, key := range privateKeys {
		unusedPrivateKeyIDs = append(unusedPrivateKeyIDs, key.ID)
	}
	return unusedPrivateKeyIDs, nil
}

func (l *Logic) clearFastlyUnusedPrivateKeys(ctx *Context) {
	for _, privateKeyID := range l.ObservedState.UnusedPrivateKeyIDs {
		ctx.Log.Info(fmt.Sprintf("attempting to delete unused private key %s", privateKeyID))
		if err := l.FastlyClient.DeletePrivateKey(&fastly.DeletePrivateKeyInput{ID: privateKeyID}); err != nil {
			// Deleting a private key has some inconsistencies on Fastly's end.
			// It is never critical to delete a private key, we only need deletion to be eventually consistent.
			// We effectively swallow the error, but notify via an info log that wont trigger a monitor.
			ctx.Log.Info(fmt.Sprintf("Failed to delete Fastly private key %s: %v. This is not critical, there are often race conditions when querying for unused private keys", privateKeyID, err))
		}
	}
}
