package main
import (
	"./checkset"
	"./futility"
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Spec struct {
	Pattern string
	Target checkset.Platform
}

type Options struct {
	Specs []Spec
	Root string
	Output string
}

func ReadSpecific(stream io.Reader) ([]Spec, error) {
	var err error
	specs := make([]Spec, 0)
	reader := bufio.NewReader(os.Stdin)
	for {
		var ln string
		ln, err = reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Print(err)
			} else {
				err = nil
			}
			break
		}
		ln = strings.TrimRight(ln, "\r\n")
		split := strings.SplitN(ln, ":", 2)
		pat := split[0]
		if pat == "" {
			continue
		}
		pspec := ""
		if len(split) > 1 {
			pspec = split[1]
		}
		var platform checkset.Platform
		platform, err = checkset.ParsePlatform(pspec)
		if err != nil {
			break
		}
		specs = append(specs, Spec { pat, platform })
	}
	return specs, err
}

func GetOptions() Options {
	root := flag.String("root", ".", "directory from which to generate the update")
	output := flag.String("output", "", "file to write checkset to; stdout if not specified")
	specify := flag.Bool("specify", false, "read file platform specifications from stdin")
	flag.Parse()
	var specs []Spec
	var err error
	if *specify {
		specs, err = ReadSpecific(os.Stdin)
		if err != nil {
			log.Print(err)
			specs = nil
		}
	}
	return Options { specs, *root, *output }
}

func GetPlatform(opts Options, name string) checkset.Platform {
	name = filepath.ToSlash(name)
	for i := range opts.Specs {
		match, _ := path.Match(opts.Specs[i].Pattern, name)
		if match {
			return opts.Specs[i].Target
		}
	}
	return checkset.AllPlatforms
}
func GetInfo(opts Options, stat futility.StatFile) checkset.CreateInfo {
	return checkset.CreateInfo {
		stat.Name,
		stat.Info.Mode(),
		GetPlatform(opts, stat.Name),
	}
}
func FilterTranslate(opts Options, in chan futility.StatFile, out chan checkset.CreateInfo) {
	for sf := range in {
	 	out <- GetInfo(opts, sf)
	}
	close(out)
}

func MakeCheckSet(opts Options) checkset.CheckSet {
	files := make(chan futility.StatFile)
	pipe := make(chan checkset.CreateInfo)
	generate := make(chan checkset.CheckSet)
	filter := func(statfile futility.StatFile) bool {
		platform := GetPlatform(opts, statfile.Name)
		return platform.OS != 0 && platform.Arch != 0
	}
	go futility.Recurse(opts.Root, filter, files)
	go FilterTranslate(opts, files, pipe)
	go checkset.Create(opts.Root, pipe, generate)
	return <-generate
}

func WriteCheckSet(opts Options, cset checkset.CheckSet) error {
	var err error
	out := os.Stdout
	if opts.Output != "" {
		out, err = os.Create(opts.Output)
		if err != nil {
			return err
		}
		defer out.Close()
	}
	return checkset.Write(cset, out)
}

func main() {
	opts := GetOptions()
	cset := MakeCheckSet(opts)
	err := WriteCheckSet(opts, cset)
	if err != nil {
		log.Print(err)
	}
}
