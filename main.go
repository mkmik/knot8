package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mkmik/multierror"
	"gopkg.in/yaml.v3"
)

type Context struct {
}

var cli struct {
	Set SetCmd `cmd help:"Set a knob."`
}

type SetCmd struct {
	Values []string `short:"v" help:"Values to set"`
	Paths  []string `optional arg:"" help:"Filenames or directories containing k8s manifests with knobs." type:"file" name:"paths"`
}

func (s *SetCmd) Run(ctx *Context) error {
	files, err := openFileArgs(s.Paths)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("cannot find any manifest in %q", s.Paths)
	}

	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()

	var (
		manifests []Manifest
		errs      []error
	)
	for _, f := range files {
		if ms, err := parseManifests(f); err != nil {
			errs = append(errs, err)
		} else {
			manifests = append(manifests, ms...)
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}

	for _, m := range manifests {
		log.Printf("--> manifest in %q, contents: %q...\n", m.file, fmt.Sprint(m.raw)[:40])
	}
	return nil
}

type Manifest struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   ObjectMetadata `json:"metadata"`

	file string
	raw  interface{}
}

type ObjectMetadata struct {
	Annotations map[string]string `json:"annotations"`
}

func parseManifests(f *os.File) ([]Manifest, error) {
	var res []Manifest
	d := yaml.NewDecoder(f)
	for {
		var i interface{}
		if err := d.Decode(&i); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		res = append(res, Manifest{file: f.Name(), raw: i})

	}
	return res, nil
}

func main() {
	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)
	err := ctx.Run(&Context{})
	ctx.FatalIfErrorf(err)
}
