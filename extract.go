package diffcopy

import (
	"os"

	"github.com/docker/containerd/fs"
)

type Extract struct {
	async bool
	dest  string
}

func (v *Extract) HandleChange(kind fs.ChangeKind, p string, fi os.FileInfo, err error) error {

	return nil
}
