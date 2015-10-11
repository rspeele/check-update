package main

import (
	"./checkset"
	"./futility"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

func EnvValue(env string) string {
	split := strings.SplitAfterN(env, "=", 2)
	if len(split) < 2 {
		return ""
	}
	return split[1]
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

func FindSauerbraten() string {
	env := os.Environ()
	best := ""
	for i := range env {
		current := strings.ToLower(env[i])
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

type ProgressWriter struct {
	written int
	lastShown int
}

func(writer *ProgressWriter) Write(p []byte) (n int, err error) {
	const MB = 1024*1024
	writer.written += len(p)
	if writer.written > writer.lastShown + MB * 5 {
		writer.lastShown = writer.written
		log.Printf("downloaded %d MB", writer.written / MB)
	}
	return len(p), nil
}

func GetSauerbraten() error {
	const InstallerPath = "Sauerbraten_Installer.exe"
	var err error
	timestamp := time.Now().Unix() * 1000
	url := fmt.Sprintf("http://downloads.sourceforge.net/project/sauerbraten/sauerbraten/2013_01_04/sauerbraten_2013_02_03_collect_edition_windows.exe?r=http%3A%2F%2Fsauerbraten.org%2F&ts=%d&use_mirror=iweb", timestamp)
	stream, err := GetHTTP(url)
	if err != nil {
		log.Print("failed to download Sauerbraten installer")
		return err
	}
	defer func () {
		stream.Close()
	}()
	output, err := os.Create(InstallerPath)
	if err != nil {
		log.Print("failed to create Sauerbraten installer file")
		return err
	}
	progress := ProgressWriter{ written:0, lastShown:0 }
	writer := io.MultiWriter(output, &progress)
	_, err = io.Copy(writer, stream)
	if err != nil {
		log.Print("failed to write installer data")
		output.Close()
		return err
	}
	output.Close()
	err = exec.Command(InstallerPath).Run()
	if err != nil {
		log.Print("error running Sauerbraten installer (try running it manually?)")
	}
	return nil
}

func RunGame(sauer string) error {
	wd, _ := os.Getwd()
	mod := path.Join(wd, "toastermod")
	exe := path.Join(mod, "toastermod_bin", "toastermod.exe")
	cmd := exec.Command(exe,
		"-k" + path.Join(mod, "toastermod"),
		"-k" + path.Join(mod, "svncompat"),
		"-q" + path.Join(mod, "userconfig"),
		"-g" + path.Join(mod, "log.txt"))
	cmd.Dir = sauer
	return cmd.Run()
}

func main() {
	var err error
	var count int
	var sauer string
	meta := flag.Bool("meta", true, "update the updater")
	flag.Parse()
	os.Remove(os.Args[0] + ".trash")
	if *meta {
		count, err = Update("http://airstrafe.com/updates/meta/", "meta.chk", path.Dir(os.Args[0]))
		if err != nil {
			goto end
		}
		if count > 0 {
			// re-launch self
			cmd := exec.Command(os.Args[0])
			cmd.Stdin = os.Stdin
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()
			go io.Copy(os.Stdout, stdout)
			go io.Copy(os.Stderr, stderr)
			cmd.Run()
			return
		}
	}
	sauer = FindSauerbraten()
	if sauer == "" {
		log.Print("you do not appear to have Sauerbraten installed")
		log.Print("downloading Sauerbraten installer, this may take a while")
		err = GetSauerbraten()
		if err != nil {
			goto end
		}
		sauer = FindSauerbraten()
	}
	count, err = Update("http://airstrafe.com/updates/toastermod/", "toastermod.chk", "toastermod")
	if err != nil {
		goto end
	}
	log.Printf("%d files required updates", count)
	err = RunGame(sauer)
end:
	if err != nil {
		log.Print(err)
		log.Print("your installation is incomplete")
	} else {
		log.Print("your installation is up to date")
	}
}