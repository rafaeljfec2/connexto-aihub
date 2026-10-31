package main

import "testing"

func TestE07ScenarioResultRequiresPassExceptCodexProgressiveRendering(t *testing.T) {
	cases := []struct {
		name       string
		result     string
		shouldFail bool
	}{
		{name: "basic_generation", result: "PASS"},
		{name: "basic_generation", result: "NOT VERIFIED", shouldFail: true},
		{name: "tool_round_trip", result: "NOT RUN — environment/provider limitation", shouldFail: true},
		{name: "codex_progressive_rendering", result: "NOT VERIFIED"},
		{name: "codex_progressive_rendering", result: "FAIL", shouldFail: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name+"/"+testCase.result, func(t *testing.T) {
			if got := e07ScenarioResultMustFail(testCase.name, testCase.result); got != testCase.shouldFail {
				t.Fatalf("scenario result failure = %t, want %t", got, testCase.shouldFail)
			}
		})
	}
}
