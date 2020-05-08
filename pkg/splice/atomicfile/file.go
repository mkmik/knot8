// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package atomicfile

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"knot8.io/pkg/splice/transform"
)

// Writer returns a io.WriteCloser that writes data to a temporary file
// which gets renamed atomically as filename upon Commit.
func Writer(filename string, perm os.FileMode) (*AtomicWriter, error) {
	out, err := ioutil.TempFile(filepath.Dir(filename), ".*~")
	if err != nil {
		return nil, err
	}
	if st, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		perm = st.Mode()
	}
	if err := os.Chmod(out.Name(), perm); err != nil {
		return nil, err
	}

	return &AtomicWriter{out, filename}, nil
}

type AtomicWriter struct {
	*os.File
	filename string
}

func (a *AtomicWriter) Close() error {
	defer os.RemoveAll(a.Name())
	return a.File.Close()
}

func (a *AtomicWriter) Commit() error {
	defer a.Close()
	return os.Rename(a.Name(), a.filename)
}

func WriteFrom(filename string, r io.Reader, perm os.FileMode) error {
	w, err := Writer(filename, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Commit()
}

// WriteFile is a drop-in replacement for ioutil.WriteFile that writes the file atomically.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	return WriteFrom(filename, bytes.NewReader(data), perm)
}

// Transform reads the content of an existing file, passes it through a transformer and writes it back atomically.
func Transform(t transform.Transformer, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w, err := Writer(filename, 0)
	if err != nil {
		return err
	}
	defer w.Close()

	if err := t.Transform(w, f); err != nil {
		return err
	}
	return w.Commit()
}
