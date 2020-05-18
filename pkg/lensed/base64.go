package lensed

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
)

// Base64Lens implements the "bas64" lens.
type Base64Lens struct{}

// Apply implements the Lens interface.
func (Base64Lens) Apply(src []byte, vals []Setter) ([]byte, error) {
	enc := base64.StdEncoding

	b, err := ioutil.ReadAll(base64.NewDecoder(enc, bytes.NewReader(src)))
	if err != nil {
		return nil, err
	}

	for _, v := range vals {
		if p := v.Pointer; p != "/" {
			return nil, fmt.Errorf("base64 lens has no structure, invalid pointer %q", p)
		}
		var err error
		b, err = v.Value.Transform(b)
		if err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	w := base64.NewEncoder(enc, &buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	w.Close()
	return buf.Bytes(), nil
}
