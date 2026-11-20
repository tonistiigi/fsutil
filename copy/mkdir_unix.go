//go:build !windows

package fs

import (
	"os"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func fixRootDirectory(p string) string {
	return p
}

func Utimes(p string, tm *time.Time) error {
	if tm == nil {
		return nil
	}

	ts, err := unix.TimeToTimespec(*tm)
	if err != nil {
		return err
	}

	timespec := []unix.Timespec{ts, ts}
	if err := unix.UtimesNanoAt(unix.AT_FDCWD, p, timespec, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.Wrapf(err, "failed to utime %s", p)
	}

	return nil
}

func snapshotDirInfo(p string) (os.FileInfo, error) {
	fi, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to stat %s", p)
	}
	if !fi.IsDir() {
		return nil, nil
	}
	return fi, nil
}

func restoreDirModTimeIfChanged(p string, prevFi os.FileInfo) error {
	curFi, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to stat %s", p)
	}
	if !curFi.IsDir() || !os.SameFile(prevFi, curFi) {
		return nil
	}
	prevMtime := StatMtime(prevFi.Sys().(*syscall.Stat_t))
	curMtime := StatMtime(curFi.Sys().(*syscall.Stat_t))
	if curMtime == prevMtime {
		return nil
	}
	timespec := []unix.Timespec{
		{Sec: utimeOmit, Nsec: utimeOmit},
		{Sec: prevMtime.Sec, Nsec: prevMtime.Nsec},
	}
	if err := unix.UtimesNanoAt(unix.AT_FDCWD, p, timespec, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			return nil
		}
		return errors.Wrapf(err, "failed to utime %s", p)
	}

	return nil
}

func Chown(p string, old *User, fn Chowner) error {
	if fn == nil {
		return nil
	}
	user, err := fn(old)
	if err != nil {
		return errors.WithStack(err)
	}
	if user != nil {
		if err := os.Lchown(p, user.UID, user.GID); err != nil {
			return err
		}
	}
	return nil
}
