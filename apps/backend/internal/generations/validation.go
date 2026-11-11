package generations

import (
	"errors"
	"strings"
	"unicode/utf8"

	"aihub/internal/execution"
	"aihub/internal/records"
)

var errInvalidTelemetry = errors.New("generations: invalid telemetry")

// ValidateTelemetry rejects IDs, events, and durations outside storage bounds.
func ValidateTelemetry(input TelemetryInput) error {
	if !records.ValidExecutionID(input.ExecutionID) || !validText(input.ProviderID, 1, 128) ||
		!validText(input.ModelID, 0, 4096) || len(input.Result.TelemetryEvents) > MaxProviderEvents {
		return errInvalidTelemetry
	}
	for _, event := range input.Result.TelemetryEvents {
		if !knownEvent(event.Kind) || event.ObservedAt.IsZero() ||
			!validText(event.ProviderID, 1, 128) || !validText(event.ModelID, 0, 4096) ||
			(event.UpstreamStatus != nil && (*event.UpstreamStatus < 100 || *event.UpstreamStatus > 599)) ||
			(event.DurationNS != nil && *event.DurationNS < 0) {
			return errInvalidTelemetry
		}
	}
	if input.FinishReason != nil && !validText(*input.FinishReason, 1, 128) {
		return errInvalidTelemetry
	}
	return nil
}

func validText(value string, minimum, maximum int) bool {
	if len(value) < minimum || len(value) > maximum || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func safeSessionID(value string) *string {
	if value == "" || len(value) > 256 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return nil
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return nil
		}
	}
	return &value
}

// SafeSessionID returns a validated bounded identifier suitable for detail storage.
func SafeSessionID(value string) *string {
	return safeSessionID(value)
}

func knownEvent(kind execution.EventKind) bool {
	switch kind {
	case execution.EventRoutingSelected, execution.EventUpstreamRequestStarted,
		execution.EventUpstreamResponseArrived, execution.EventFirstOutput,
		execution.EventGenerationFinished:
		return true
	default:
		return false
	}
}
