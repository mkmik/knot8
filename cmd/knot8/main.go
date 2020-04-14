// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/mkmik/multierror"
)

type Context struct {
}

var cli struct {
	Set SetCmd `cmd help:"Set a knob."`
}

type SetCmd struct {
	Values []string `short:"v" help:"Value to set. Format: field:value"`
	Paths  []string `optional arg:"" help:"Filenames or directories containing k8s manifests with knobs." type:"file" name:"paths"`
}

func (s *SetCmd) Run(ctx *Context) (err error) {
	paths := s.Paths

	if len(s.Paths) == 0 {
		stdin, err := slurpStdin()
		paths = []string{stdin}
		defer func() {
			if err == nil {
				if f, err := os.Open(stdin); err != nil {
					log.Println(err)
				} else {
					io.Copy(os.Stdout, f)
					f.Close()
				}
			}
		}()
	}

	files, err := openFiles(paths)
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
		manifests []*Manifest
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

	knobs, err := parseKnobs(manifests)
	if err != nil {
		return err
	}

	for _, f := range s.Values {
		c := strings.SplitN(f, ":", 2)
		if len(c) != 2 {
			errs = append(errs, fmt.Errorf("bad -v format %q, missing ':'", f))
			continue
		}
		if err := setKnob(knobs, c[0], c[1]); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}

	return nil
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
