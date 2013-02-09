// create.go - creation of a CheckSet from files

package checkset

import (
	"os"
	"path/filepath"
)

type CreateInfo struct {
	Name string
	Mode os.FileMode
	Target Platform
}

func Create(root string, files chan CreateInfo, result chan CheckSet) {
	cset := make(CheckSet)
	for filestat := range files {
		if filestat.Target.OS == 0 || filestat.Target.Arch == 0 || filestat.Mode.IsDir() {
			continue
		}
		hash, err := HashFile(filestat.Name)
		if err != nil {
			break
		}
		rel, err := filepath.Rel(root, filestat.Name)
		if err != nil {
			rel = filestat.Name
		}
		cset[rel] = CheckInfo {
			filestat.Target,
			filestat.Mode,
			hash,
		}
	}
	result <- cset
}