package diffcopy

import (
	"os"

	"github.com/docker/containerd/fs"
)

type Processor interface {
	HandleChange(fs.ChangeKind, string, os.FileInfo, error) error
	Close() error
}
