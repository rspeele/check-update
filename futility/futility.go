package futility

import "path/filepath"
import "os"
import "io"
import "log"

var selfPath string
func RecordSelfPath() error {
	var err error
	selfPath, err = filepath.Abs(os.Args[0])
	return err
}
func IsSelfPath(path string) bool {
	abs, _ := filepath.Abs(path)
	return selfPath != "" && abs == selfPath
}

func PathExists(path string, mode os.FileMode) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode() & mode != 0
}
func DirectoryExists(path string) bool {
	return PathExists(path, os.ModeDir)
}
func FileExists(path string) bool {
	return PathExists(path, os.ModePerm)
}

func CopyFile(from string, to string, mode os.FileMode) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	err = out.Chmod(mode)
	return err
}

func redir(path string, from string, to string) (string, error) {
	rel, err := filepath.Rel(from, path)
	if err != nil {
		return "", err
	}
	return filepath.Join(to, rel), nil
}

const noRecursiveCopy = os.ModeSymlink |
	os.ModeDevice |
	os.ModeNamedPipe |
	os.ModeSocket |
	os.ModeTemporary

func recursiveCopy(path string, info os.FileInfo, from string, to string) error {
	mode := info.Mode()
	target, err := redir(path, from, to)
	if err != nil {
		return err
	}
	if mode&noRecursiveCopy != 0 {
		return filepath.SkipDir
	}
	dir := mode.IsDir()
	if dir {
		err = os.Mkdir(target, mode)
		if err != nil {
			log.Print(err)
			return filepath.SkipDir
		}
		return nil
	}
	err = CopyFile(path, target, mode)
	if err != nil {
		log.Print(err)
	}
	return nil
}
func RecursiveCopy(from string, to string) {
	dir := filepath.Dir(filepath.Clean(from))
	walker := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			err = recursiveCopy(path, info, dir, to)
		}
		return err
	}
	filepath.Walk(from, walker)
}

type StatFile struct {
	Name string
	Info os.FileInfo
}
func NoFilter(sf StatFile) bool {
	return true
}
func Recurse(root string, filter func(StatFile) bool, files chan StatFile) {
	walker := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			sf := StatFile { path, info }
			if filter(sf) {
				files <- StatFile { path, info }
			} else {
				return filepath.SkipDir
			}
		}
		return err
	}
	filepath.Walk(root, walker)
	close(files)
}

// try hard to create a file with MODE at PATH. creates any missing
// directories in PATH and will try to move un-writable files out of
// the way if necessary.
func Create(path string, mode os.FileMode) (io.WriteCloser, error) {
	var err error
	err = os.MkdirAll(filepath.Dir(path), 0744)
	if err != nil {
		return nil, err
	}
	var wr *os.File
	for tries := 0; tries < 2; tries++ {
		wr, err = os.OpenFile(path, os.O_CREATE | os.O_WRONLY, mode)
		// sometimes, can't overwrite a file, but can move it out of the way
		if err != nil {
			trash := path + ".trash"
			os.Remove(trash)
			rename := os.Rename(path, trash)
			if rename == nil {
				continue
			} else {
				break
			}
		}
	}
	if err == nil && wr != nil {
		wr.Chmod(mode)
	}
	return wr, err

}
