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

### In-place edits

You can even mutate the file in-place!
Yes, I know, it sounds outrageous but you might learn to stop worrying and love the knot8 merge feature.

```sh
$ knot8 set -f testdata/m1.yaml foo=hola
$ git diff
```
```diff
diff --git testdata/m1.yaml testdata/m1.yaml
index ea3a8fd..04a2143 100644
--- testdata/m1.yaml
+++ testdata/m1.yaml
@@ -32,7 +32,7 @@ spec:
         env:
           # Voilá!
           - name: FOO
-            value: bar # See ^^^ field.knot8.io/foo
+            value: hola # See ^^^ field.knot8.io/foo
         volumeMounts:
          - name: config
            mountPath: /cfg
@@ -48,4 +48,4 @@ metadata:
   annotations:
     field.knot8.io/foo: /data/foo
 data:
-  foo: bar # See ^^^ field.knot8.io/foo
+  foo: hola # See ^^^ field.knot8.io/foo
```
