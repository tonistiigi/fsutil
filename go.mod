module github.com/tonistiigi/fsutil

go 1.25.0

require (
	github.com/Microsoft/go-winio v0.6.2
	github.com/containerd/continuity v0.5.0
	github.com/moby/patternmatcher v0.6.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/pkg/errors v0.9.1
	github.com/planetscale/vtprotobuf v0.6.0
	github.com/stretchr/testify v1.11.1
	github.com/tonistiigi/dchapes-mode v0.0.0-20250318174251-73d941a28323
	golang.org/x/sync v0.19.0
	golang.org/x/sys v0.41.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/containerd/log v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jstemmer/go-junit-report/v2 v2.1.0 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

tool (
	github.com/jstemmer/go-junit-report/v2
	github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto
	google.golang.org/protobuf/cmd/protoc-gen-go
)
