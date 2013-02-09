package main

import (
	"./checkset"
	"./futility"
	"errors"
	"os/exec"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

func EnvValue(env string) string {
	split := strings.SplitAfterN(env, "=", 2)
	if len(split) < 2 {
		return ""
	}
	return split[1]
}

func FindSauerbraten() string {
	env := os.Environ()
	best := ""
	for i := range env {
		current := strings.ToLower(env[i])
		println(current)
		if strings.HasPrefix(current, "programfiles(x86)=") {
			best = env[i]
			break
		}
		if strings.HasPrefix(current, "programfiles=") {
			best = env[i]
		}
	}
	if best == "" {
		return ""
	}
	split := strings.SplitAfterN(best, "=", 2)
	if len(split) < 2 {
		return ""
	}
	sauerbraten := path.Join(split[1], "Sauerbraten")
	if futility.DirectoryExists(sauerbraten) {
		return sauerbraten
	}
	return ""
}

func IsTesseract(dir string) bool {
	return futility.DirectoryExists(path.Join(dir, "tesseract")) &&
		futility.FileExists(path.Join(dir, "tesseract.bat"))
}

func FindTesseract() string {
	child := "tesseract"
	if IsTesseract(child) {
		return child
	}
	test := "."
	lasttest := ""
	for i := 0; i < 5; i++ {
		if IsTesseract(test) {
			return test
		}
		test = path.Join("..", test)
		if test == lasttest {
			break
		}
		lasttest = test
	}
	return child
}

var MissingPackages = errors.New("you need the packages directory from an install of sauerbraten (see sauerbraten.org)")

func RestorePackages(tesseract string) {
	sauerbraten := FindSauerbraten()
	pkgs := "packages"
	tesspack := path.Join(tesseract, pkgs)
	tessexist := futility.DirectoryExists(tesspack)
	if sauerbraten == "" {
		if !tessexist {
			panic(MissingPackages)
		}
		return
	}
	sauerpack := path.Join(sauerbraten, pkgs)
	sauerexist := futility.DirectoryExists(sauerpack)
	if tessexist {
		if !sauerexist {
			log.Printf("restoring previously moved packages from %s to %s", tesspack, sauerpack)
			futility.RecursiveCopy(tesspack, sauerpack)
			log.Print("done")
		}
	} else if sauerexist {
		if !tessexist {
			log.Printf("copying packages from %s to %s", sauerpack, tesspack)
			futility.RecursiveCopy(sauerpack, tesspack)
			log.Print("done")
		}
	}
}

func GetHTTP(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		// accept no HTTP errors
		resp.Body.Close()
		return nil, errors.New(resp.Status)
	}
	return resp.Body, err
}

func GetFrom(base string) checkset.Retriever {
	return func(name string) (io.ReadCloser, error) {
		return GetHTTP(base + name)
	}
}

func DescribeReason(reason int) string {
	switch reason {
	case checkset.Valid:
		return "valid (this is a bug, but probably harmless)"
	case checkset.Missing:
		return "missing"
	case checkset.HashMismatch:
		return "outdated"
	case checkset.TypeMismatch:
		return "wrong file type"
	case checkset.PermMismatch:
		return "permissions"
	}
	return "unknown reason (this is a bug)"
}

func ShowBad(in chan checkset.BadFile, count *int, out chan checkset.BadFile) {
	*count = 0
	for bad := range in {
		why := DescribeReason(bad.Reason)
		log.Printf("%-40s %s", bad.Remote, why)
		out <- bad
		*count++
	}
	close(out)
}

func Update(source, update, local string) (int, error) {
	log.Printf("downloading checkset %s", update)
	stream, err := GetFrom(source)(update)
	if err != nil {
		return 0, err
	}
	cset, err := checkset.Read(stream)
	if err != nil {
		return 0, err
	}
	log.Printf("verifying %d files in %s", len(cset), local)
	bad := make(chan checkset.BadFile)
	pipe := make(chan checkset.BadFile)
	erc := make(chan error)
	var count int
	go checkset.Verify(local, cset, bad)
	go ShowBad(bad, &count, pipe)
	go checkset.Apply(local, GetFrom(source), pipe, erc)
	for err = range erc {
		log.Print(err)
	}
	return count, err
}

func main() {
	count, err := Update("http://silentunicorn.com/updates/meta/", "meta.chk", path.Dir(os.Args[0]))
	if err != nil {
		log.Print(err)
		return
	}
	if count > 0 {
		// re-launch self
		if false {
		cmd := exec.Command(os.Args[0])
		stdin, _ := cmd.StdinPipe()
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		go io.Copy(stdin, os.Stdin)
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
		cmd.Run()
		return
		}
	}
	tesseract := FindTesseract()
	count, err = Update("http://silentunicorn.com/updates/tesseract/", "tesseract.chk", tesseract)
	if err != nil {
		log.Print(err)
		return
	}
	log.Printf("%d files required updates", count)
	RestorePackages(tesseract)
}