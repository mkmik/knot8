// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/mkmik/multierror"
)

type Context struct {
}

var cli struct {
	Set  SetCmd  `cmd help:"Set a knob."`
	Get  GetCmd  `cmd help:"Get the value of knob."`
	Info InfoCmd `cmd help:"Show available knobs."`
}

type CommonFlags struct {
	Paths []string `name:"filename" short:"f" help:"Filenames or directories containing k8s manifests with knobs." type:"file"`
}

type SetCmd struct {
	CommonFlags
	Values []string `arg:"" help:"Value to set. Format: field=value"`
}

func (s *SetCmd) Run(ctx *Context) (err error) {
	knobs, printStdin, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			printStdin()
		}
	}()

	var errs []error
	for _, f := range s.Values {
		c := strings.SplitN(f, "=", 2)
		if len(c) != 2 {
			errs = append(errs, fmt.Errorf("bad -v format %q, missing '='", f))
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

type GetCmd struct {
	CommonFlags
	Field string `arg:"" help:"Field to get."`
}

func (s *GetCmd) Run(ctx *Context) error {
	knobs, _, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}

	values, err := getKnob(knobs, s.Field)
	if err != nil {
		return err
	}
	for _, v := range values {
		s, err := renderKnobValue(v)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", s)
	}
	return nil
}

// renderKnobValue reads a knob value from the source stream and reformats it so it displays nicely
// in the get command output. It preserves the value formatting from the source yaml but re-indents it
// and drops the comment from the source.
func renderKnobValue(k knobValue) (string, error) {
	filename := k.ptr.Manifest.file
	r, err := readFileRunes(filename)
	if err != nil {
		return "", err
	}

	v := string(k.loc.slice(r))
	v = strings.TrimSuffix(v, "\n")
	c := strings.SplitN(v, "\n", 2)
	if len(c) == 2 {
		style, body := c[0], c[1]
		i := strings.Index(style, "#")
		if i > 0 {
			style = style[0:i]
		}
		v = fmt.Sprintf("%s\n%s", style, reindent(body, 2))
	}
	return fmt.Sprintf("%s:%d: %s", filename, k.line, v), nil
}

type InfoCmd struct {
	CommonFlags
}

func (s *InfoCmd) Run(ctx *Context) error {
	knobs, _, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}

	fmt.Println("Knobs:")
	var names []string
	for k := range knobs {
		names = append(names, k)
	}

	sort.Strings(names)
	for _, k := range names {
		fmt.Printf("  %s\n", k)
	}

	return nil
}

// openKnobs returns a map of knobs defined in the set of files referenced by the path arguments (see openFiles).
// It also returns a printStdin callback, meant to be called before exiting successfully in order
// to print out the content of the (possibly modified) stream when using knot8 in "pipe" mode.
func openKnobs(pathArgs []string) (knobs map[string]Knob, printStdin func(), err error) {
	paths, printStdin, err := wrapStdin(pathArgs)
	if err != nil {
		return nil, nil, err
	}

	files, err := openFiles(paths)
	if err != nil {
		return nil, nil, err
	}
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("cannot find any manifest in %q", paths)
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
		return nil, nil, multierror.Join(errs)
	}

	knobs, err = parseKnobs(manifests)
	return knobs, printStdin, err
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
