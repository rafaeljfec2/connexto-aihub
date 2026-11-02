//go:build !linux

package support

func E07CodexProcessActive() (bool, error) {
	return E07CodexProcessUnsupported()
}
