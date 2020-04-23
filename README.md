# knot8

Define and manipulate "knobs" in K8s manifests.

## Example

Imagine some YAML manifest you download from upstream looks like:

```sh
$ cat testdata/m1.yaml
```
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
  annotations:
    field.knot8.io/foo: /spec/template/spec/containers/~{"name":"app"}/env/~{"name":"FOO"}/value
spec:
... # trimmed for readability, see testdata/m1.yaml in this repo for the full example
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
  annotations:
    field.knot8.io/foo: /data/foo
data:
  foo: bar # See ^^^ field.knot8.io/foo
```

### Change some knobs

```sh
$ knot8 set <testdata/m1.yaml foo=hola >/tmp/c1.yaml
$ diff -u testdata/m1.yaml /tmp/c1.yaml
```
```diff
--- testdata/m1.yaml	2020-04-15 10:08:26.000000000 +0200
+++ /tmp/c1.yaml	2020-04-23 18:18:42.000000000 +0200
@@ -32,7 +32,7 @@
         env:
           # Voilá!
           - name: FOO
-            value: bar # See ^^^ knot8.io/foo
+            value: hola # See ^^^ knot8.io/foo
         volumeMounts:
          - name: config
            mountPath: /cfg
@@ -48,4 +48,4 @@
   annotations:
     field.knot8.io/foo: /data/foo
 data:
-  foo: bar # See ^^^ field.knot8.io/foo
+  foo: hola # See ^^^ field.knot8.io/foo
```

### Take values from files

```sh
$ knot8 set <testdata/m1.yaml --from values.yaml >/tmp/c1.yaml
```

Where values.yaml file is a YAML manifest file of any kind containing knot8 annotated fields.

### In-place edits

You can even mutate the file in-place!
Yes, I know, it sounds outrageous but you might learn to stop worrying and love the knot8 merge feature (see next section).

```sh
$ knot8 set -f testdata/m1.yaml foo=hola
$ git status -s -b
```
```
## master...origin/master
 M testdata/m1.yaml
```

### Manual edits

As you can see from the diffs `knot8 set` only changes the values themselves.
You could make those changes manually (or with some other tool) if you so wish!

```sh
$ vim testdata/m1.yaml
```

or:

```sh
$ sed 's/bar/hola/' -i testdata/m1.yaml
```

Usually `knot8` will do a better job finding all the fields, but in principle they are just simple edits
to your files, no magic voodoo.

### Merge upstream changes

You can upgrade a manifest to a new version while retaining all the local changes made to the fields.

You can get the new version of the manifest from any HTTP server:

```sh
$ knot8 pull -f testdata/m1.yaml https://github.com/some/app/releases/download/v1.2.3/app.yaml
```

Or any source supported by the [go-getter](https://github.com/hashicorp/go-getter#url-format) url-format:

```sh
$ knot8 pull -f testdata/m1.yaml https://github.com/some/app//app.yaml?ref=dev
```

The algorithm is a 3-way merge between:

a. your current file.
b. the new upstream.
c. the common baseline.

The common baseline can be provided explicitly, but usually you'll rely on the original file having
a `knot8.io/base` annotation with a snapshot of the original values that will later become useful as a baseline.
