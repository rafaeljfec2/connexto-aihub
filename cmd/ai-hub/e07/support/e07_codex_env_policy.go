package support

// e07CanaryEnvVar enables the isolated Codex canary. The canary is opt-in
// because it requires the exact validated CLI release and a stable global Codex
// tree; otherwise it would make the default test suite depend on the host. When
// enabled it still fails closed instead of relaxing any assertion.
const E07CanaryEnvVar = "AIHUB_E07_CANARY"
