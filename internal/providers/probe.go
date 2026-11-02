package providers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"aihub/internal/config"
)

const probeTimeout = 10 * time.Second

// ProbeEndpoint checks the model endpoint of an OpenCode Go configuration with
// the same request the startup connectivity check uses. Errors never include
// the credential.
func ProbeEndpoint(ctx context.Context, endpoint, credential string) error {
	client, err := NewClient(config.ProviderOpenCodeGo{BaseURL: endpoint, APIKey: credential}, nil)
	if err != nil {
		return err
	}

	defer client.CloseIdle()

	probeContext, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	response, err := client.ListModels(probeContext)
	if err != nil {
		return err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("connectivity check returned status %d", response.StatusCode)
	}

	return nil
}
