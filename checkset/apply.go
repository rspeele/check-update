// apply.go - apply a resolution to failed CheckSet verification

package checkset

import (
	"../futility"
	"errors"
	"io"
	"os"
	"path"
)

// function which returns an open stream for reading a file, when
// given its remote relative path.
type Retriever func(string) (io.ReadCloser, error)

var ExistingSpecial = errors.New("don't want to replace a not-file with a file")

func Resolve(root string, bad BadFile, get Retriever) error {
	// use root argument instead of bad.Local to allow updating to
	// a different path than was checked; might be useful later
	local := path.Join(root, bad.Remote)
	switch bad.Reason {
	case Missing, HashMismatch:
		read, err := get(bad.Remote)
		if err != nil {
			return err
		}
		defer read.Close()
		write, err := futility.Create(local, bad.Info.Mode)
		if err != nil {
			return err
		}
		defer write.Close()
		_, err = io.Copy(write, read)
		return err
	case TypeMismatch:
		return ExistingSpecial
	case PermMismatch:
		return os.Chmod(local, bad.Info.Mode)
	}
	return nil
}

func Apply(root string, get Retriever, bads chan BadFile, errs chan error) {
	var err error
	for bad := range bads {
		err = Resolve(root, bad, get)
		if err != nil {
			errs <- err
		}
	}
	close(errs)
}