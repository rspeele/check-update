// platform.go - get and compare checkset.Platforms

package checkset

import (
	"errors"
	"runtime"
	"strings"
)

func parseFlags(repr string, resolve func(string) uint) (uint, error) {
	var err error
	complement := false
	if strings.HasPrefix(repr, "^") {
		repr = repr[1:]
		complement = true
	}
	split := strings.Split(repr, "|")
	var flags uint = 0
	for i := range split {
		ored := flags | resolve(split[i])
		if split[i] != "" && ored == flags {
			err = errors.New("invalid or redundant pattern " + split[i])
		}
		flags = ored;
	}
	if complement {
		flags = ^flags
	}
	return flags, err
}

func parseSingleOS(name string) uint {
	switch name {
	case "":
		return AllOSes
	case "windows":
		return Windows
	case "unix", "linux", "darwin", "openbsd", "freebsd":
		return Unix
	}
	return 0
}

func ParseOS(repr string) (OperatingSystem, error) {
	os, err := parseFlags(repr, parseSingleOS)
	return OperatingSystem(os), err
}

func CurrentOS() OperatingSystem {
	os := OperatingSystem(parseSingleOS(runtime.GOOS))
	if os == 0 {
		os = AllOSes
	}
	return os
}

func parseSingleArch(repr string) uint {
	switch repr {
	case "":
		return AllArches
	case "386":
		return I386
	case "amd64":
		return AMD64
	case "arm":
		return ARM
	}
	return 0
}
func ParseArch(repr string) (Architecture, error) {
	os, err := parseFlags(repr, parseSingleArch)
	return Architecture(os), err
}

func CurrentArch() Architecture {
	arch := Architecture(parseSingleArch(runtime.GOARCH))
	if arch == 0 {
		arch = AllArches
	}
	return arch
}

// parse a string representing a platform
// examples
// unix
// :386
// !windows:!arm
// :!arm,amd64
func ParsePlatform(spec string) (Platform, error) {
	bisect := strings.Split(spec, ":")
	lspec := bisect[0]
	rspec := ""
	if len(bisect) > 1 {
		rspec = bisect[1]
	}
	os, oerr := ParseOS(lspec)
	arch, aerr := ParseArch(rspec)
	err := oerr
	if err == nil {
		err = aerr
	}
	return Platform {
		os,
		arch,
	}, err
}

func CurrentPlatform() Platform {
	return Platform {
		CurrentOS(),
		CurrentArch(),
	}
}

func MatchPlatform(host, target Platform) bool {
	return host.OS & target.OS != 0 && host.Arch & target.Arch != 0
}