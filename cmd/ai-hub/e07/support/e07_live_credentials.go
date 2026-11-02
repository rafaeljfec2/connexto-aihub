package support

import (
	"aihub/internal/config"
)

const E07CredentialPreflightEnvVar = "AIHUB_E07_CREDENTIAL_PREFLIGHT"

func E07GatewayCredential(environment []string) string {
	return EnvironmentMap(environment)[config.EnvGatewayAPIKey]
}
