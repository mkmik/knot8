// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package yamled

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// quote quotes a string into a yaml string.
// It tries to preserve original quotation style, when it's likely to be intentional.
// In particular, if the original value had to be quoted (e.g. a number) and the new value
// doesn't have to be quoted, then the quotes will be dropped. OTOH, if the original value
// didn't have to be quoted in the first place, we'll make sure the new value is also quoted,
// just to avoid frustrating the user who intentionally quoted a string that didn't have to be quoted.
// If the user didn't intentionally quote the string, we're not making the original file style any
// worse than it already was.
func quote(value, old string, indent int) (string, error) {
	indent += 2
	if len(old) > 0 {
		q := old[0]
		if q == '"' || q == '\'' {
			reEncoded, err := yamlRoundTrip(old, indent)
			if err != nil {
				return "", err
			}
			if reEncoded[0] != q {
				if q == '"' {
					return jsonMarshalString(value)
				}
				return yamlStringTrySingleQuoted(value, indent)
			}
		}
	}

	return yamlString(value, indent)
}

func jsonMarshalString(value interface{}) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// yamlRoundTrip decodes and a string from YAML and reencodes it into YAML.
func yamlRoundTrip(str string, indent int) (string, error) {
	var parsed string
	if err := yaml.Unmarshal([]byte(str), &parsed); err != nil {
		return "", err
	}
	return yamlString(parsed, indent)
}

// yamlString returns a string quoted with yaml rules.
func yamlString(value string, indent int) (string, error) {
	if value == "" {
		return "", nil
	}
	if min := 2; indent < min {
		return "", fmt.Errorf("yamlString indent must be at least %d", min)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(value); err != nil {
		return "", err
	}

	s := buf.String()
	s = s[:len(s)-1] // strip trailing newline emitted by yaml marshaling.
	return reindent(s, indent-2), nil
}

// reindent reindents a yaml multiline string by positive indentation delta.
//
// For some reason, the go-yaml library artificially limits the indentation to 2-10 range
// https://github.com/go-yaml/yaml/issues/501
func reindent(s string, indent int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > 2 {
		for i := 1; i < len(lines); i++ {
			if len(lines[i]) > 0 {
				lines[i] = fmt.Sprintf("%s%s", strings.Repeat(" ", indent), lines[i])
			}
		}
	}
	return strings.Join(lines, "\n")
}

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// yamlStringTrySingleQuoted will return a single quoted YAML string, unless
// it's impossible to encode it as such (e.g. because it contains non-printable chars),
// in that case it will return whatever encoding yamlString picks.
func yamlStringTrySingleQuoted(s string, indent int) (string, error) {
	if !isPrintable(s) {
		return yamlString(s, indent)
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''")), nil
}
