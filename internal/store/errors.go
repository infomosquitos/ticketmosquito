package store

import "errors"

// ErrNotFound is returned when a catalog row is missing.
var ErrNotFound = errors.New("not found")
