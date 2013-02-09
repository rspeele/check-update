// types.go - CheckSet types

package checkset

import (
	"fmt"
	"os"
)

type OperatingSystem uint8
const (
	Unix = 1<<iota
	Windows
	AllOSes = 0xff
)

type Architecture uint8
const (
	I386 = 1<<iota
	AMD64
	ARM
	AllArches = 0xff
)

type Platform struct {
	OS OperatingSystem
	Arch Architecture
}

var AllPlatforms = Platform {
	AllOSes,
	AllArches,
}

type CheckInfo struct {
	Target Platform
	Mode os.FileMode
	Hash [HashSize]byte
}

type CheckSet map[string] CheckInfo

func Show(cset CheckSet) {
	for k, v := range cset {
		fmt.Printf("%-30s %000o %x\n", k, v.Mode, v.Hash)
	}
}