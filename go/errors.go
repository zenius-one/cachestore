package cachestore

import "errors"

// ErrInvalidTTL is returned by Set and SetNX for a non-positive ttl.
var ErrInvalidTTL = errors.New("cachestore: ttl must be positive")
