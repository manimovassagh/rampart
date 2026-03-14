package store

import "errors"

// Sentinel errors returned by store implementations.
var (
	ErrDuplicateKey = errors.New("duplicate key")
	ErrNotFound     = errors.New("not found")
	ErrDefaultOrg   = errors.New("cannot delete the default organization")
	ErrBuiltinRole  = errors.New("cannot modify a builtin role")
)
