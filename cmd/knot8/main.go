// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/mkmik/multierror"
)

type Context struct {
}

var cli struct {
	Set   SetCmd   `cmd help:"Set a knob."`
	Get   GetCmd   `cmd help:"Get the value of knob."`
	Merge MergeCmd `cmd help:"Merge a new version from upstream."`
	Info  InfoCmd  `cmd help:"Show available knobs."`
}

type CommonFlags struct {
	Paths []string `name:"filename" short:"f" help:"Filenames or directories containing k8s manifests with knobs." type:"file"`
}

type Setter struct {
	Field string
	Value string
}

func (s *Setter) UnmarshalText(in []byte) error {
	c := strings.SplitN(string(in), "=", 2)
	if len(c) != 2 {
		return fmt.Errorf("bad -v format %q, missing '='", in)
	}
	s.Field, s.Value = c[0], c[1]
	return nil
}

type SetCmd struct {
	CommonFlags
	Values []Setter `arg:"" help:"Value to set. Format: field=value"`
	Format string   `name:"format" short:"o" help:"If empty, the changes are performed in-place in the input yaml; Otherwise a patch is produced in a given format. Available formats: overlay, jsonnet."`
}

func (s *SetCmd) Run(ctx *Context) error {
	knobs, commit, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}

	var errs []error
	for _, f := range s.Values {
		if err := setKnob(knobs, f.Field, f.Value); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}

	switch s.Format {
	case "":
		if err := commit(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("format %q not implemented yet", s.Format)
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
	file := k.ptr.Manifest.source.file
	filename := file.name
	r := file.buf

	v := string(k.loc.slice(r))
	c := strings.SplitN(v, "\n", 2)
	if len(c) == 2 {
		style, body := c[0], c[1]
		i := strings.Index(style, "#")
		if i > 0 {
			style = style[0:i]
		}
		v = fmt.Sprintf("%s\n%s", style, reindent(body, 2))
	}

	if k.ptr.Manifest.source.fromStdin {
		filename = "-"
	}
	return fmt.Sprintf("%s:%d: %s", filename, k.line, v), nil
}

type MergeCmd struct {
	CommonFlags
	Upstream []string `optional arg:"" help:"Filename or URL of the next version of the manifest(s). Collections of files can be fetched via URLs by wrapping them into tar/zip balls."`
}

func (s *MergeCmd) Run(ctx *Context) error {
	// This impl is just a quick hack to show a POC;
	// TODO(mkm): write a real impl

	if len(s.Upstream) > 0 {
		return fmt.Errorf("only merge with stdin implemented")
	}

	if len(s.Paths) == 0 {
		return fmt.Errorf("-f required")
	}

	knobsL, _, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}

	knobsU, commit, err := openKnobs(s.Upstream)
	if err != nil {
		return err
	}

	for _, n := range knobNames(knobsU) {
		values, err := getKnob(knobsL, n)
		if err != nil {
			return err
		}
		log.Printf("GOT knob %q value %q from upstream", n, values[0].value)

		setKnob(knobsU, n, values[0].value)
	}

	return commit()
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
	for _, k := range knobNames(knobs) {
		fmt.Printf("  %s\n", k)
	}

	return nil
}

// openKnobs returns a map of knobs defined in the set of files referenced by the path arguments (see openFiles).
// It also returns a printStdin callback, meant to be called before exiting successfully in order
// to print out the content of the (possibly modified) stream when using knot8 in "pipe" mode.
func openKnobs(paths []string) (knobs map[string]Knob, commit func() error, err error) {
	fromStdin := false

	if len(paths) == 0 {
		fromStdin = true
		stdin, err := slurpStdin()
		if err != nil {
			return nil, nil, err
		}
		paths = []string{stdin}
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
		manifests Manifests
		errs      []error
	)
	for _, f := range files {
		s, err := newShadowFile(f)
		if err != nil {
			errs = append(errs, err)
		} else if ms, err := parseManifests(s, fromStdin); err != nil {
			errs = append(errs, err)
		} else {
			manifests = append(manifests, ms...)
		}
	}
	if errs != nil {
		return nil, nil, multierror.Join(errs)
	}

	knobs, err = parseKnobs(manifests)
	return knobs, manifests.Commit, err
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
