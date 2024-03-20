module github.com/tonistiigi/fsutil/bench

go 1.20

require (
	github.com/containerd/continuity v0.4.1
	github.com/docker/docker v24.0.9+incompatible
	github.com/pkg/errors v0.9.1
	github.com/tonistiigi/fsutil v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.1.0
)

require (
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/Microsoft/hcsshim v0.8.17 // indirect
	github.com/containerd/containerd v1.5.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/moby/patternmatcher v0.5.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/runc v1.0.0-rc93 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	golang.org/x/sys v0.1.0 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
)

replace github.com/tonistiigi/fsutil => ../
