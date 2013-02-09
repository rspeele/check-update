// verify.go - verification of a CheckSet against files on disk

package checkset

import (
	"path"
	"os"
)

// reasons a file could be bad
const (
	Valid = iota
	Missing
	HashMismatch
	TypeMismatch
	PermMismatch
)

type BadFile struct {
	Remote string
	Local string
	Info CheckInfo
	Reason int
}

func TestFile(local string, info CheckInfo) int {
	fi, err := os.Lstat(local)
	switch {
	case err != nil:
		return Missing
	case fi.Mode() & os.ModeType != 0:
		return TypeMismatch
	case !CheckHash(local, info.Hash):
		return HashMismatch
	case fi.Mode() & info.Mode != info.Mode:
		return PermMismatch
	}
	return Valid
}

// send file paths failing verification to failed
func Verify(root string, cset CheckSet, failed chan BadFile) {
	platform := CurrentPlatform()
	for file, info := range cset {
		if !MatchPlatform(platform, info.Target) {
			continue // skip files not targeted for this platform
		}
		local := path.Join(root, file)
		reason := TestFile(local, info)
		if reason > Valid {
			failed <- BadFile {
				file,
				local,
				info,
				reason,
			}
		}
	}
	close(failed)
}