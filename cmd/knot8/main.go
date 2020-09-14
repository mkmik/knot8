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
	"knot8.io/pkg/lensed"
)

const (
	Knot8file = "Knot8file"
)

type Context struct {
}

var cli struct {
	Set    SetCmd    `cmd:"" help:"Set a field value."`
	Cat    CatCmd    `cmd:"" help:"Like set but always output to stdout"`
	Values ValuesCmd `cmd:"" help:"Show available fields."`
	Diff   DiffCmd   `cmd:"" help:"Show the values different from the original."`
	Pull   PullCmd   `cmd:"" help:"Pull and merge a new version from upstream."`
	Lint   LintCmd   `cmd:"" help:"Check that the manifests follow the knot8 rules."`
	Schema SchemaCmd `cmd:"" help:"Emit the schema. Can also be used to generate a Knot8file from an inline annotated manifest set."`

	Version kong.VersionFlag `name:"version" help:"Print version information and quit"`
}

type CommonFlags struct {
	Paths []string `name:"filename" short:"f" help:"Filenames or directories containing k8s manifests with fields." type:"file"`
}

type CommonSchemaFlags struct {
	Schema string `name:"schema" help:"File containing field definitions. Used to augment the field definitions present inline in the resource annotations. The file format mirrors the format of real K8s resources, but shall only contain apiVersion,kind,metadata name, namespace and field annotations."`
}

func (c *CommonSchemaFlags) AfterApply() error {
	if c.Schema == "" {
		_, err := os.Stat(Knot8file)
		if err == nil {
			c.Schema = Knot8file
		}
	}
	return nil
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

type CatCmd struct {
	SetCmd
}

func (c *CatCmd) Run(ctx *Context) error {
	c.Stdout = true
	return c.SetCmd.Run(ctx)
}

type SetCmd struct {
	CommonFlags
	CommonSchemaFlags

	Values []Setter `optional:"" arg:"" help:"Value to set. Format: field=value or field=@filename, where a leading @ can be escaped with a backslash."`
	From   []string `name:"from" type:"file" help:"Read values from one or more files."`
	Freeze bool     `name:"freeze" help:"Save current values to knot8.io/original."`
	Stdout bool     `name:"stdout" help:"Output to stdout and never update files in-place"`
}

func (s *SetCmd) Run(ctx *Context) error {
	// if Knot8file exists, use it as a source of default values.
	_, err := os.Stat(Knot8file)
	if err == nil {
		s.From = append([]string{Knot8file}, s.From...)
	}

	manifestSet, err := openFields(s.Paths, s.Schema)
	if err != nil {
		return err
	}

	// if outputing to stdout instead of inline (either via --stdout, or because of the cat command),
	// rename all filenames to "-" causing them to be treated as stdio upon commit.
	if s.Stdout {
		for _, m := range manifestSet.Manifests {
			m.source.file.name = "-"
		}
	}

	values := s.Values
	if len(s.From) > 0 {
		fromValues, err := settersFromFiles(s.From)
		if err != nil {
			return err
		}
		values = append(fromValues, values...)
	}

	batch := manifestSet.Fields.NewEditBatch()
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

	if s.Freeze {
		if err := freeze(manifestSet); err != nil {
			return err
		}
	}

	return manifestSet.Manifests.Commit()
}

func settersFromFiles(paths []string) ([]Setter, error) {
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
			all[k] = v
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
	manifestSet, err := openFields(s.Paths, "")
	if err != nil {
		return err
	}
	d, err := diff(manifestSet)
	if err != nil {
		return err
	}
	return yaml.NewEncoder(os.Stdout).Encode(d)
}

func diff(manifestSet *ManifestSet) (map[string]string, error) {
	o, err := findOriginal(manifestSet)
	if err != nil {
		return nil, err
	}

	dirty := map[string]string{}
	for n, k := range manifestSet.Fields {
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

func freeze(ms *ManifestSet) error {
	for _, m := range ms.Manifests {
		if _, ok := m.Metadata.Annotations[originalAnno]; ok {
			if err := updateOriginalAnno(m.source, ms.Fields); err != nil {
				return err
			}
		}
	}
	return nil
}

func updateOriginalAnno(src manifestSource, fields map[string]Field) error {
	path := fmt.Sprintf("/metadata/annotations/%s", strings.ReplaceAll(originalAnno, "/", "~1"))
	body, err := renderOriginalAnnoBody(fields)
	if err != nil {
		return err
	}
	edits := []lensed.Mapping{
		{fmt.Sprintf("~(yamls)/%d%s", src.streamPos, path), string(body)},
	}
	f := src.file
	b, err := lensed.Apply(f.buf, edits)
	if err != nil {
		return err
	}
	f.buf = b
	return nil
}

func renderOriginalAnnoBody(fields map[string]Field) ([]byte, error) {
	values := map[string]string{}
	for n, k := range fields {
		kv, err := k.GetAll()
		if err != nil {
			return nil, err
		}
		values[n] = kv[0].value
	}
	return yaml.Marshal(&values)
}

type PullCmd struct {
	CommonFlags
	CommonSchemaFlags
	Upstream string `arg:"" help:"Upstream file/URL." type:"file"`
}

func (s *PullCmd) Run(ctx *Context) error {
	// quick&dirty 3-way merge that deals with only one current and one upstream file
	// (which can contain multiple manifests).
	if len(s.Paths) > 1 {
		return fmt.Errorf("pull/merge with %d files currently not supported", len(s.Paths))
	}

	manifestSetC, err := openFields(s.Paths, s.Schema)
	if err != nil {
		return err
	}
	d, err := diff(manifestSetC)
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

	manifestSetU, err := openFields([]string{upstream.Name()}, s.Schema)
	if err != nil {
		return err
	}
	batch := manifestSetU.Fields.NewEditBatch()
	for n, v := range d {
		batch.Set(n, v)
	}
	if err := batch.Commit(); err != nil {
		return err
	}

	msC, msU := manifestSetC.Manifests, manifestSetU.Manifests
	msC[0].source.file.buf = msU[0].source.file.buf

	return manifestSetC.Manifests.Commit()
}

type ValuesCmd struct {
	CommonFlags
	CommonSchemaFlags

	NamesOnly bool   `short:"k" help:"Print only field names and not their values."`
	Field     string `arg:"" optional:"" help:"Print the value of one specific field"`
}

func (s *ValuesCmd) Run(ctx *Context) error {
	manifestSet, err := openFields(s.Paths, s.Schema)
	if err != nil && !(isNotUniqueValueError(err) && (s.NamesOnly || s.Field != "")) {
		return err
	}

	if s.NamesOnly {
		for _, n := range manifestSet.Fields.Names() {
			fmt.Printf("%s\n", n)
		}
		return nil
	} else if s.Field != "" {
		v, err := manifestSet.Fields.GetValue(s.Field)
		if err != nil {
			return err
		}
		fmt.Println(v)
		return nil
	} else {
		b, err := renderOriginalAnnoBody(manifestSet.Fields)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(b)
		return err
	}
}

type LintCmd struct {
	CommonFlags
	CommonSchemaFlags
}

func (s *LintCmd) Run(ctx *Context) error {
	manifestSet, err := openFields(s.Paths, s.Schema)
	if err != nil {
		return err
	}

	if err := checkFields(manifestSet.Fields); err != nil {
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

func checkFields(fields Fields) error {
	var errs []error
	for _, n := range fields.Names() {
		values, err := fields.GetAll(n)
		if err != nil {
			errs = append(errs, err)
		} else if !checkFieldValues(values) {
			var vs []string
			for _, v := range values {
				vs = append(vs, v.value)
			}
			errs = append(errs, fmt.Errorf("values pointed by field %q are not unique (%q)", n, vs))
		}
	}
	if errs != nil {
		return errNotUniqueValue{multierror.Join(errs)}
	}
	return nil
}

type SchemaCmd struct {
	CommonFlags
	CommonSchemaFlags
}

func (s *SchemaCmd) Run(ctx *Context) error {
	manifestSet, err := openFields(s.Paths, s.Schema)
	if err != nil {
		return err
	}

	enc := yaml.NewEncoder(os.Stdout)
	for _, m := range manifestSet.Manifests {
		if len(m.Metadata.Annotations) > 0 {
			enc.Encode(m)
		}
	}

	return nil
}

// openFields returns a map of fields defined in the set of files referenced by the path arguments (see openFiles).
// It also returns a printStdin callback, meant to be called before exiting successfully in order
// to print out the content of the (possibly modified) stream when using knot8 in "pipe" mode.
func openFields(paths []string, schema string) (*ManifestSet, error) {
	var (
		manifests Manifests
		fields    Fields
	)
	if len(paths) == 0 {
		paths = []string{"-"}
	}

	filenames, err := expandPaths(paths)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("cannot find any manifest in %q", paths)
	}

	var (
		errs []error
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
		return nil, multierror.Join(errs)
	}

	fields, err = parseFields(manifests)
	if err != nil {
		return nil, err
	}

	if schema != "" {
		s, err := newShadowFile(schema)
		if err != nil {
			return nil, err
		}
		ms, err := parseManifests(s)
		if err != nil {
			return nil, err
		}
		ms = ms.Intersect(manifests)
		manifests.MergeAnnotations(ms)
		ext, err := parseFields(ms)
		if err != nil {
			return nil, err
		}
		if err := ext.Rebase(manifests); err != nil {
			return nil, err
		}
		fields.MergeSchema(ext)
	}

	err = checkFields(fields)
	// let the caller decide whether the validation error is fatal

	return &ManifestSet{Fields: fields, Manifests: manifests}, err
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
