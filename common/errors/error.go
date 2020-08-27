package errors

import "errors"

// Common errors.
var (
	NotFound = errors.New("error: Not Found")
	NotMatch = errors.New("error: Not Match")
)
