// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

/*
Package yamled implements helpers for in-place editing of YAML sources.

The in-place editing itself is delegated to the knot8.io/pkg/splice package.

This package adds functions to derive a splice.Selection from a yaml.Node,
and a YAML-specific value quoting implementation.

*/
package yamled
