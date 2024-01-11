package e5e

import "fmt"

// InvalidEntrypointError is returned if the given entrypoint did not get registered before invoking [Start].
type InvalidEntrypointError struct{ Entrypoint string }

func (e InvalidEntrypointError) Error() string {
	return fmt.Sprintf("entrypoint %q does not exist", e.Entrypoint)
}
