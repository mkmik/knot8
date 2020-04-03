package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
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
	fmt.Printf("setting files %q, knobs: %q\n", s.Paths, s.Values)
	files, err := openFileArgs(s.Paths)
	if err != nil {
		return err
	}
	fmt.Printf("got %v\n", files)
	if len(files) == 0 {
		return fmt.Errorf("cannot find any manifest in %q", s.Paths)
	}

	defer func() {
		for _, f := range files {
			log.Printf("closing %q\n", f.Name())
			f.Close()
		}
	}()

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
