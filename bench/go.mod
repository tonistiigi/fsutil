module github.com/tonistiigi/fsutil/bench

go 1.13

require (
	github.com/Microsoft/hcsshim v0.8.17 // indirect
	github.com/containerd/containerd v1.5.2 // indirect
	github.com/containerd/continuity v0.1.0
	github.com/docker/docker v20.10.7+incompatible
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/tonistiigi/fsutil v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

replace github.com/tonistiigi/fsutil => ../
