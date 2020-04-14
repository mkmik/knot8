# knot8
define and manipulate "knobs" in K8s manifests

## Example

```sh
$ knot8 set testdir/m1.yaml -v foo:hola
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
