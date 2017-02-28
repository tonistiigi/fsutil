package fsutil

import (
	"os"

	"github.com/docker/containerd/fs"
)

// Hardlinks validates that all targets for links were

type Hardlinks struct {
	seenFiles map[string]struct{}
}

func (v *Hardlinks) HandleChange(kind fs.ChangeKind, p string, fi os.FileInfo, err error) error {

	return nil
}
