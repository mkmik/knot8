// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hashicorp/go-getter"
	"github.com/mkmik/multierror"
	"gopkg.in/yaml.v3"
)

type Context struct {
}

var cli struct {
	Set    SetCmd    `cmd:"" help:"Set a field value."`
	Values ValuesCmd `cmd:"" help:"Show available fields."`
	Diff   DiffCmd   `cmd:"" help:"Show the values different from the original."`
	Pull   PullCmd   `cmd:"" help:"Pull and merge a new version from upstream."`
	Lint   LintCmd   `cmd:"" help:"Check that the manifests follow the knot8 rules."`

	Version kong.VersionFlag `name:"version" help:"Print version information and quit"`
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

	if strings.HasPrefix(s.Value, "@") {
		b, err := ioutil.ReadFile(strings.TrimPrefix(s.Value, "@"))
		if err != nil {
			return err
		}
		s.Value = string(b)
	} else if strings.HasPrefix(s.Value, `\@`) {
		s.Value = strings.TrimPrefix(s.Value, `\`)
	}

	return nil
}

type SetCmd struct {
	CommonFlags
	Values []Setter `optional:"" arg:"" help:"Value to set. Format: field=value or field=@filename, where a leading @ can be escaped with a backslash."`
	From   []string `name:"from" type:"file" help:"Read values from one or more files. The values will be read from not8 annotated k8s resources."`
	Format string   `name:"format" short:"o" help:"If empty, the changes are performed in-place in the input yaml; Otherwise a patch is produced in a given format. Available formats: overlay, jsonnet."`
}

func (s *SetCmd) Run(ctx *Context) error {
	knobs, commit, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}

	values := s.Values
	if len(s.From) > 0 {
		fromValues, err := settersFromFiles(s.From)
		if err != nil {
			return err
		}
		values = append(fromValues, values...)
	}

	batch := knobs.NewEditBatch()
	var errs []error
	for _, f := range values {
		if err := batch.Set(f.Field, f.Value); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return multierror.Join(errs)
	}
	if err := batch.Commit(); err != nil {
		return err
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

func settersFromFiles(paths []string) ([]Setter, error) {
	knobs, _, err := openKnobs(paths)
	if err != nil {
		return nil, err
	}

	if err := checkKnobs(knobs); err != nil {
		return nil, err
	}

	var res []Setter
	for _, n := range knobs.Names() {
		values, err := knobs.GetAll(n)
		if err != nil {
			return nil, err
		}
		res = append(res, Setter{n, values[0].value})
	}

	simple, err := openSimplifiedValues(paths)
	if err != nil {
		return nil, err
	}
	res = append(res, simple...)

	return res, nil
}

func openSimplifiedValues(paths []string) ([]Setter, error) {
	var (
		res  []Setter
		errs []error
		all  = map[string]string{}
	)
	for _, path := range paths {
		values, err := parseSimplifiedValues(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for k, v := range values {
			if old, ok := all[k]; ok && old != v {
				errs = append(errs, errNotUniqueValue{fmt.Errorf("value in field %q is not unique", k)})
			} else {
				all[k] = v
			}
		}
	}
	if errs != nil {
		return nil, multierror.Join(errs)
	}
	for k, v := range all {
		res = append(res, Setter{k, v})
	}
	return res, nil
}

func parseSimplifiedValues(path string) (map[string]string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.APIVersion != "" || m.Kind != "" {
		return nil, nil
	}

	var values map[string]string
	if err := yaml.Unmarshal(b, &values); err != nil {
		return nil, err
	}
	return values, nil
}

type DiffCmd struct {
	CommonFlags
}

func (s *DiffCmd) Run(ctx *Context) error {
	knobs, _, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}
	d, err := diff(knobs)
	if err != nil {
		return err
	}
	return yaml.NewEncoder(os.Stdout).Encode(d)
}

func diff(knobs map[string]Knob) (map[string]string, error) {
	o, err := findOriginal(knobs)
	if err != nil {
		return nil, err
	}

	dirty := map[string]string{}
	for n, k := range knobs {
		values, err := k.GetAll()
		if err != nil {
			return nil, err
		}
		if v := values[0].value; o[n] != v {
			dirty[n] = v
		}
	}
	return dirty, nil
}

type PullCmd struct {
	CommonFlags
	Upstream string `arg:"" help:"Upstream file/URL." type:"file"`
}

func (s *PullCmd) Run(ctx *Context) error {
	// quick&dirty 3-way merge that deals with only one current and one upstream file
	// (which can contain multiple manifests).
	if len(s.Paths) > 1 {
		return fmt.Errorf("pull/merge with %d files currently not supported", len(s.Paths))
	}

	knobsC, commit, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}
	d, err := diff(knobsC)
	if err != nil {
		return err
	}

	upstream, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	opt := func(c *getter.Client) (err error) {
		c.Pwd, err = os.Getwd()
		return
	}
	if err := getter.GetFile(upstream.Name(), s.Upstream, opt); err != nil {
		return err
	}

	knobsU, _, err := openKnobs([]string{upstream.Name()})
	if err != nil {
		return err
	}
	batch := knobsU.NewEditBatch()
	for n, v := range d {
		batch.Set(n, v)
	}
	if err := batch.Commit(); err != nil {
		return err
	}

	msC := allManifests(knobsC)
	msU := allManifests(knobsU)
	msC[0].source.file.buf = msU[0].source.file.buf

	return commit()
}

type ValuesCmd struct {
	CommonFlags

	NamesOnly bool   `short:"k" help:"Print only field names and not their values."`
	Field     string `arg:"" optional:"" help:"Print the value of one specific field"`
}

func (s *ValuesCmd) Run(ctx *Context) error {
	knobs, _, err := openKnobs(s.Paths)
	if err != nil && !(isNotUniqueValueError(err) && (s.NamesOnly || s.Field != "")) {
		return err
	}

	if s.NamesOnly {
		for _, n := range knobs.Names() {
			fmt.Printf("%s\n", n)
		}
		return nil
	} else if s.Field != "" {
		v, err := knobs.GetValue(s.Field)
		if err != nil {
			return err
		}
		fmt.Println(v)
		return nil
	} else {
		values := map[string]string{}
		for n, k := range knobs {
			kv, err := k.GetAll()
			if err != nil {
				return err
			}
			values[n] = kv[0].value
		}
		return yaml.NewEncoder(os.Stdout).Encode(&values)
	}
}

type LintCmd struct {
	CommonFlags
}

func (s *LintCmd) Run(ctx *Context) error {
	knobs, _, err := openKnobs(s.Paths)
	if err != nil {
		return err
	}

	if err := checkKnobs(knobs); err != nil {
		return err
	}

	return nil
}

type errNotUniqueValue struct{ err error }

func (e errNotUniqueValue) Error() string { return e.err.Error() }
func (e errNotUniqueValue) Unwrap() error { return e.err }

func isNotUniqueValueError(err error) bool {
	var u errNotUniqueValue
	return errors.As(err, &u)
}

func checkKnobs(knobs Knobs) error {
	var errs []error
	for _, n := range knobs.Names() {
		values, err := knobs.GetAll(n)
		if err != nil {
			errs = append(errs, err)
		} else if !checkKnobValues(values) {
			errs = append(errs, fmt.Errorf("values pointed by field %q are not unique", n))
		}
	}
	if errs != nil {
		return errNotUniqueValue{multierror.Join(errs)}
	}
	return nil
}

// openKnobs returns a map of knobs defined in the set of files referenced by the path arguments (see openFiles).
// It also returns a printStdin callback, meant to be called before exiting successfully in order
// to print out the content of the (possibly modified) stream when using knot8 in "pipe" mode.
func openKnobs(paths []string) (knobs Knobs, commit func() error, err error) {
	if len(paths) == 0 {
		paths = []string{"-"}
	}

	filenames, err := expandPaths(paths)
	if err != nil {
		return nil, nil, err
	}
	if len(filenames) == 0 {
		return nil, nil, fmt.Errorf("cannot find any manifest in %q", paths)
	}

	var (
		manifests Manifests
		errs      []error
	)
	for _, f := range filenames {
		s, err := newShadowFile(f)
		if err != nil {
			errs = append(errs, err)
		} else if ms, err := parseManifests(s); err != nil {
			errs = append(errs, err)
		} else {
			manifests = append(manifests, ms...)
		}
	}
	if errs != nil {
		return nil, nil, multierror.Join(errs)
	}

	knobs, err = parseKnobs(manifests)
	if err != nil {
		return nil, nil, err
	}

	err = checkKnobs(knobs)
	// let the caller decide whether the validation error is fatal

	return knobs, manifests.Commit, err
}

func main() {
	ctx := kong.Parse(&cli,
		kong.UsageOnError(),
		kong.Vars{
			"version": "0.0.1",
		},
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)
	err := ctx.Run(&Context{})
	ctx.FatalIfErrorf(err)
}
