//+build freebsd solaris

package fs

import (
	"os"
	"io"

	"github.com/pkg/errors"
)

func copyFile(source, target string) error {
        src, err := os.Open(source)
        if err != nil {
                return errors.Wrapf(err, "failed to open source %s", source)
        }
        defer src.Close()
        tgt, err := os.Create(target)
        if err != nil {
                return errors.Wrapf(err, "failed to open target %s", target)
        }
        defer tgt.Close()

        return copyFileContent(tgt, src)
}

func copyFileContent(dst, src *os.File) error {
        _, err := io.Copy(dst, src)
        if(err != nil) {
                return err
        }
        return nil
}
