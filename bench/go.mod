module github.com/tonistiigi/fsutil/bench

go 1.25.0

require (
	github.com/containerd/continuity v0.5.0
	github.com/pkg/errors v0.9.1
	github.com/tonistiigi/fsutil v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.19.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/moby/patternmatcher v0.6.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/tonistiigi/fsutil => ../
