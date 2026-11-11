package routing_test

import (
	"os"
	"testing"

	"aihub/internal/routing"
)

func TestInitAutoGate0FromEnv(t *testing.T) {
	prevFlag := routing.AutoGate0Enforced
	prevEnv := os.Getenv(routing.EnvAutoGate0Enforced)
	t.Cleanup(func() {
		routing.AutoGate0Enforced = prevFlag
		if prevEnv == "" {
			_ = os.Unsetenv(routing.EnvAutoGate0Enforced)
		} else {
			_ = os.Setenv(routing.EnvAutoGate0Enforced, prevEnv)
		}
	})

	routing.AutoGate0Enforced = false
	_ = os.Setenv(routing.EnvAutoGate0Enforced, "true")
	routing.InitAutoGate0FromEnv()
	if !routing.AutoGate0Enforced {
		t.Fatal("expected AutoGate0Enforced true after env=true")
	}

	_ = os.Setenv(routing.EnvAutoGate0Enforced, "false")
	routing.InitAutoGate0FromEnv()
	if routing.AutoGate0Enforced {
		t.Fatal("expected AutoGate0Enforced false after env=false")
	}
}
