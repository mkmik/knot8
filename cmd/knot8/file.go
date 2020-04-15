// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/mkmik/multierror"
)

// openFiles opens all files referenced by the paths slice.
// If a path points to a directory, openFiles will open all *.yaml files contained in it.
func openFiles(paths []string) ([]*os.File, error) {
	var (
		files []*os.File
		errs  []error
	)

	for _, p := range paths {
		fs, err := openManifestsAt(p)
		if err != nil {
			errs = append(errs, err)
		} else {
			files = append(files, fs...)
		}
	}

	if errs != nil {
		return nil, multierror.Join(errs)
	}
	return files, nil
}

// openManifestsAt will open the file p and return it if it's a simple file,
// otherwise, if it's a directory, it will open all the K8s manifest files contained in it (see manifestsInDir).
func openManifestsAt(p string) ([]*os.File, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	if st, err := f.Stat(); err != nil {
		return nil, err
	} else if st.IsDir() {
		paths, err := manifestsInDir(f)
		if err != nil {
			return nil, err
		}
		return openFiles(paths)
	}

	return []*os.File{f}, nil
}

// manifestsInDir returns all potential K8s manifest files in a directory.
func manifestsInDir(dir *os.File) ([]string, error) {
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	var res []string
	for _, n := range names {
		if ok, err := matchExts(n, "yaml", "yml", "json"); err != nil {
			return nil, err
		} else if ok {
			res = append(res, filepath.Join(dir.Name(), n))
		}
	}
	return res, nil
}

// wrapStdin handles the case when pathArgs represent stdin.
//
// Stdin is slurped into a tmp file and wrapStdin returns a new path slice
// containing that file. It also returns a function meant to be called
// to print out the contents of this (possibly modified) tmp file.
// This function is a no-op in case the pathArgs do not represent a stdin.
func wrapStdin(pathArgs []string) (paths []string, printStdin func(), err error) {
	printStdin = func() {}
	if len(pathArgs) == 0 {
		stdin, err := slurpStdin()
		if err != nil {
			return nil, nil, err
		}
		pathArgs = []string{stdin}
		printStdin = func() {
			if f, err := os.Open(stdin); err != nil {
				log.Println(err)
			} else {
				io.Copy(os.Stdout, f)
				f.Close()
			}
		}
	}
	return pathArgs, printStdin, nil
}

func matchExts(filename string, exts ...string) (bool, error) {
	for _, e := range exts {
		if ok, err := filepath.Match(fmt.Sprintf("*.%s", e), filename); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}

// slurpStdin reads stdin fully and saves into a temporary file, whose path name is returned.
func slurpStdin() (string, error) {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Fprintf(os.Stderr, "(reading manifests from standard input; hit ctrl-c if this is not what you wanted)\n")
	}

	tmp, err := ioutil.TempFile("", "stdin")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(tmp, os.Stdin)
	if err != nil {
		return "", err
	}
	return tmp.Name(), nil
}
