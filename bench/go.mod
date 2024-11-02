module github.com/tonistiigi/fsutil/bench

go 1.21

require (
	github.com/containerd/continuity v0.4.4
	github.com/docker/docker v27.3.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/tonistiigi/fsutil v0.0.0-00010101000000-000000000000
	golang.org/x/sync v0.8.0
)

require (
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20240806141605-e8a1dd7889d6 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/moby/patternmatcher v0.5.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.26.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gotest.tools/v3 v3.0.3 // indirect
)

replace github.com/tonistiigi/fsutil => ../
