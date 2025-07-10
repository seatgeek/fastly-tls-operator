package fastlycertificatesync

// RuntimeConfig contains the runtime configuration for the FastlyCertificateSync controller
type RuntimeConfig struct {
	// Configuration fields can be added here as needed
	HackFastlyCertificateSyncLocalReconciliation bool
}

// Config wraps the runtime configuration
type Config struct {
	RuntimeConfig
}
