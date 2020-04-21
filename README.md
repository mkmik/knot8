# knot8
define and manipulate "knobs" in K8s manifests

## Example

```sh
$ cat testdir/m1.yaml
```
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
  annotations:
    field.knot8.io/foo: /spec/template/spec/containers/~{"name":"app"}/env/~{"name":"FOO"}/value
spec:
... # removed in this readme
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
  annotations:
    field.knot8.io/foo: /data/foo
data:
  foo: bar # 8< field.knot8.io/foo
```
```sh
$ knot8 set -f testdir/m1.yaml foo=hola
$ git diff
```
```diff
diff --git testdir/m1.yaml testdir/m1.yaml
index ea3a8fd..04a2143 100644
--- testdir/m1.yaml
+++ testdir/m1.yaml
@@ -32,7 +32,7 @@ spec:
         env:
           # Voilá!
           - name: FOO
-            value: bar # 8< knot8.io/foo
+            value: hola # 8< knot8.io/foo
         volumeMounts:
          - name: config
            mountPath: /cfg
@@ -48,4 +48,4 @@ metadata:
   annotations:
     field.knot8.io/foo: /data/foo
 data:
-  foo: bar # 8< field.knot8.io/foo
+  foo: hola # 8< field.knot8.io/foo
```
