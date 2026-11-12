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
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value || !validText(value, 1, 256) {
		return nil
	}
	return &value
}

// SafeSessionID returns a validated bounded identifier suitable for detail storage.
func SafeSessionID(value string) *string {
	return safeSessionID(value)
}

// SafeClientApp normalizes and bounds an originating client application name to 128 UTF-8 bytes.
func SafeClientApp(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || !validText(trimmed, 1, len(trimmed)) {
		return nil
	}
	if len(trimmed) > 128 {
		end := 128
		for end > 0 && !utf8.RuneStart(trimmed[end]) {
			end--
		}
		trimmed = strings.TrimSpace(trimmed[:end])
		if trimmed == "" {
			return nil
		}
	}
	return &trimmed
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
