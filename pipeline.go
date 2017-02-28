package fsutil

import (
	"os"

	"github.com/docker/containerd/fs"
)

type HandleChangeFn func(fs.ChangeKind, string, os.FileInfo, error) error

type Processor interface {
	HandleChange(fs.ChangeKind, string, os.FileInfo, error) error
	Close() error
}
