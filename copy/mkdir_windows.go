//go:build windows

package fs

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

const (
	containerAdministratorSidString = "S-1-5-93-2-1"
)

func fixRootDirectory(p string) string {
	if len(p) == len(`\\?\c:`) {
		if os.IsPathSeparator(p[0]) && os.IsPathSeparator(p[1]) && p[2] == '?' && os.IsPathSeparator(p[3]) && p[5] == ':' {
			return p + `\`
		}
	}
	return p
}

func Utimes(p string, tm *time.Time) error {
	info, err := os.Lstat(p)
	if err != nil {
		return errors.Wrap(err, "fetching file info")
	}
	if tm != nil && info.Mode()&os.ModeSymlink == 0 {
		if err := os.Chtimes(p, *tm, *tm); err != nil {
			return errors.Wrap(err, "changing times")
		}
	}
	return nil
}

func longPathUTF16(path string) ([]uint16, error) {
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\??\`) || strings.HasPrefix(path, `\\.\`) {
		return windows.UTF16FromString(path)
	}
	in, err := windows.UTF16FromString(path)
	if err != nil {
		return nil, err
	}
	n := uint32(len(in))
	var full []uint16
	for {
		buf := make([]uint16, n)
		r, err := windows.GetFullPathName(&in[0], n, &buf[0], nil)
		if err != nil {
			return nil, err
		}
		if r < n {
			full = buf[:r]
			break
		}
		n = r
	}
	abs := windows.UTF16ToString(full)
	if len(abs) < 248 {
		return windows.UTF16FromString(abs)
	}
	if strings.HasPrefix(abs, `\\`) {
		return windows.UTF16FromString(`\\?\UNC\` + abs[2:])
	}
	return windows.UTF16FromString(`\\?\` + abs)
}

func snapshotDirInfo(p string) (os.FileInfo, error) {
	fullPath, err := longPathUTF16(p)
	if err != nil {
		return nil, errors.Wrap(err, "converting to utf16 path")
	}
	h, err := windows.CreateFile(
		&fullPath[0],
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to stat %s", p)
	}
	f := os.NewFile(uintptr(h), p)
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat %s", p)
	}
	if !fi.IsDir() {
		return nil, nil
	}
	return fi, nil
}

func restoreDirModTimeIfChanged(p string, prevFi os.FileInfo) error {
	rawPrev, ok := prevFi.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return nil
	}
	curFi, err := snapshotDirInfo(p)
	if err != nil || curFi == nil || !os.SameFile(prevFi, curFi) {
		return err
	}
	if rawCur, ok := curFi.Sys().(*syscall.Win32FileAttributeData); ok && rawCur.LastWriteTime == rawPrev.LastWriteTime {
		return nil
	}
	fullPath, err := longPathUTF16(p)
	if err != nil {
		return errors.Wrap(err, "converting to utf16 path")
	}
	h, err := windows.CreateFile(
		&fullPath[0],
		windows.FILE_READ_ATTRIBUTES|windows.FILE_WRITE_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
			return nil
		}
		return errors.Wrap(err, "opening handle to restore mtime")
	}
	f := os.NewFile(uintptr(h), p)
	defer f.Close()

	recheckFi, err := f.Stat()
	if err != nil {
		return errors.Wrap(err, "reading handle attributes")
	}
	if !recheckFi.IsDir() || !os.SameFile(prevFi, recheckFi) {
		return nil
	}
	ft := windows.Filetime{
		LowDateTime:  rawPrev.LastWriteTime.LowDateTime,
		HighDateTime: rawPrev.LastWriteTime.HighDateTime,
	}
	if err := windows.SetFileTime(h, nil, nil, &ft); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return nil
		}
		return errors.Wrap(err, "restoring mtime")
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
	var userSIDstring string
	if user != nil && user.SID != "" {
		userSIDstring = user.SID
	}
	if userSIDstring == "" {
		userSIDstring = containerAdministratorSidString

	}

	sidPtr, err := syscall.UTF16PtrFromString(userSIDstring)
	if err != nil {
		return errors.Wrap(err, "converting to utf16 ptr")
	}
	var userSID *windows.SID
	if err := windows.ConvertStringSidToSid(sidPtr, &userSID); err != nil {
		return errors.Wrap(err, "converting to windows SID")
	}
	var dacl *windows.ACL
	newEntries := []windows.EXPLICIT_ACCESS{
		{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeValue: windows.TrusteeValueFromSID(userSID),
			},
		},
	}
	newAcl, err := windows.ACLFromEntries(newEntries, dacl)
	if err != nil {
		return fmt.Errorf("adding acls: %w", err)
	}

	// Copy file ownership and ACL
	// We need SeRestorePrivilege and SeTakeOwnershipPrivilege in order
	// to restore security info on a file, especially if we're trying to
	// apply security info which includes SIDs not necessarily present on
	// the host.
	privileges := []string{winio.SeRestorePrivilege, seTakeOwnershipPrivilege}
	err = winio.RunWithPrivileges(privileges, func() error {
		if err := windows.SetNamedSecurityInfo(
			p, windows.SE_FILE_OBJECT,
			windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
			userSID, nil, newAcl, nil); err != nil {

			return err
		}
		return nil
	})

	return err
}
