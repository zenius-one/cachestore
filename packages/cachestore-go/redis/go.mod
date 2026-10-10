module github.com/zenius-one/cachestore/packages/cachestore-go/redis

go 1.27.1

// The requirement on the core module (github.com/zenius-one/cachestore/packages/cachestore-go) is
// deliberately absent until that module has a tagged release: the go.work in the repository root
// resolves it locally, and `go` insists on fetching a required version that does not exist yet.
// After packages/cachestore-go/v0.1.0 is tagged, pin it, then add a changeset for this module:
//
//	GOWORK=off go get github.com/zenius-one/cachestore/packages/cachestore-go@v0.1.0


require (
	github.com/alicebob/miniredis/v2 v2.40.0
	github.com/redis/go-redis/v9 v9.23.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)
