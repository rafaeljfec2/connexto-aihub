package routing

import (
	"os"
	"strconv"
	"strings"
)

// EnvAutoGate0Enforced enables the all-ten Gate 0 kill-switch for aihub/auto when true.
const EnvAutoGate0Enforced = "AIHUB_AUTO_GATE0_ENFORCED"

// InitAutoGate0FromEnv applies AIHUB_AUTO_GATE0_ENFORCED when set to a valid boolean.
func InitAutoGate0FromEnv() {
	raw := strings.TrimSpace(os.Getenv(EnvAutoGate0Enforced))
	if raw == "" {
		return
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return
	}
	AutoGate0Enforced = parsed
}
