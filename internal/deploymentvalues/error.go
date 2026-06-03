package deploymentvalues

import "errors"

// ErrInvalidTemplate signals a user-caused template error (bad syntax or execution failure).
// Errors not wrapped with this sentinel should be treated as server-side failures.
var ErrInvalidTemplate = errors.New("invalid template")
