package store

import "errors"

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")
