package fsutil

import (
	"os"
)

type HandleChangeFn func(ChangeKind, string, os.FileInfo, error) error

type Processor interface {
	HandleChange(ChangeKind, string, os.FileInfo, error) error
	Close() error
}
