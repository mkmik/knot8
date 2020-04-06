// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
	"github.com/mkmik/multierror"
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

	for _, m := range manifests {
		log.Printf("--> manifest in %q, contents: %#v\n", m.file, m)
	}

	knobs, err := parseKnobs(manifests)
	if err != nil {
		return err
	}

	log.Printf("------------")
	log.Printf("knobs: %v\n", knobs)
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
