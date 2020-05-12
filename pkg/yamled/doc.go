// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

/*
Package yamled implements helpers for in-place editing of YAML sources.

The editing is performed by a golang.org/x/text/transform.Transformer implementation
configured with one or more editing operations.

Editing operations are defined as string replacements over selections covering YAML nodes in the YAML source.

Selections are constructing from *yaml.Node value that can be obtained somehow (e.g. manually or via the YAML JSONPointer or YAML JSONPath libraries).

*/
package yamled
