// hash.go - hashing files

package checkset

import (
	"crypto/sha1"
	"errors"
	"io"
	"os"
)

const HashSize = 20

func HashFile(path string) ([HashSize]byte, error) {
	var hash [HashSize]byte
	file, err := os.Open(path)
	if err != nil {
		return hash, err
	}
	defer file.Close()
	sha := sha1.New()
	io.Copy(sha, file)
	sum := sha.Sum(nil)
	if copy(hash[:], sum) != HashSize {
		return hash, errors.New("hash was wrong size")
	}
	return hash, err
}

func CheckHash(path string, hash [HashSize]byte) bool {
	check, err := HashFile(path)
	return err == nil && hash == check
}