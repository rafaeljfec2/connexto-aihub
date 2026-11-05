package providers

import (
	"errors"
	"regexp"
)

// TypeOpenCodeGo is the adapter type of the OpenCode Go integration.
const TypeOpenCodeGo = "opencode-go"

// Connexto Proxy integration identifiers and defaults.
const (
	ConnextoProxyID              = "connexto-proxy-cli-llms"
	TypeConnextoProxy            = "connexto-proxy"
	DefaultConnextoProxyEndpoint = "http://127.0.0.1:9090/v1"
)

var (
	ErrInvalid          = errors.New("invalid provider configuration")
	ErrNoCredential     = errors.New("provider has no credential configured")
	ErrProbeUnavailable = errors.New("provider connectivity validation is not supported")
	idPattern           = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
)

const (
	maxNameLength       = 100
	maxCredentialLength = 4096
)
