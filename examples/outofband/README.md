# Out of band

While the main UX of the knot8 experiment is based on inline annotations and in-place updates,
there is an alternative way to use the same tool that might better suite those who want to
keep the source files pristine.

```bash
$ mkdir playground
$ cd playground
```

Let's first download an example YAML from the internet:

```bash
$ wget https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/application/deployment.yaml
$ cat deployment.yaml
apiVersion: apps/v1 # for versions before 1.9.0 use apps/v1beta2
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  selector:
    matchLabels...
```

Now, let's imagine you wanted to update one specfic location of that YAML file; let's say you want to update
its metadata name field. Since there could be many resources in your manifest set, we need to precisely address
the resource you want to reference. Instead of using complex commandline flags to list appVersion, kind, name and
namespace, you just write a _normal_ K8s resource and you declare the knot8 field as an annotation of this dummy resource:

```bash
cat >Knot8file <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    field.knot8.io/appName: /metadata/name
EOF
```

Now you told knot8 about the _schema_ of your manifest set and you can operate on _fields.
For example you can see the current values of all the declared fields. As we didn't override the field values,
we'll see the value present in the deployment.yaml manifest as pointed to by the `/metadata/name` JSON Pointer:


```bash
$ knot8 values -f .
appName: nginx-deployment
```

The `Knot8file` is read in by default; you can save the schema in another file, you'd have to pass `--schema` explicitly.

Now, let's apply an override:

```bash
$ knot8 set -f . --stdout appName=mytest
apiVersion: apps/v1 # for versions before 1.9.0 use apps/v1beta2
kind: Deployment
metadata:
  name: mytest
spec:
  selector:
    matchLabels....
```

(The `--stdout` flag instructs knot8 to not mutate the manifest set in-place. No worries, there a syntax sugar for the non-inplace workflow, bear with me for a moment)

What happened here is that knot updated the `appName` field with the user specified value `mytest`.
The output is a legal manifest set, so it's possible to further process it with knot8:

```
$ knot8 set -f . --stdout appName=mytest | knot8 values
appName: mytest
```

A more useful and slightly more complicated example involves overriding the image name. Let's add an `appImage`
field that points to the image string of the nginx container inside the deployment spec template:

```bash
cat >Knot8file <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    field.knot8.io/appName: /metadata/name
    field.knot8.io/appImage: /spec/template/spec/containers/~{"name":"nginx"}/image
EOF
```

```bash
$ knot8 values -f .
appImage: nginx:1.14.2
appName: nginx-deployment
$ knot8 set -f . --stdout appImage=bitnami/nginx:1.14.2
...
      containers:
      - name: nginx
        image: bitnami/nginx:1.14.2
...
```

Setting those values on the commandline is tedious, let's save them in a file:

```bash
$ knot8 set -f . --stdout appImage=bitnami/nginx:1.14.2 >staging.yaml
```

And use it later:

```bash
$ knot8 set -f . --stdout --from staging.yaml
```

You can even inline the default values into the `Knot8file`:

```
$ cat >Knot8file <<EOF
appImage: bitnami/nginx:1.14.2
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    field.knot8.io/appName: /metadata/name
    field.knot8.io/appImage: /spec/template/spec/containers/~{"name":"nginx"}/image
EOF
```

And you can use `cat` as a shorthand for `set --stdout`:

```
$ knot8 cat -f .
...
      containers:
      - name: nginx
        image: bitnami/nginx:1.14.2
...
```

The `knot8` command will come with helpers to maintain such `Knot8file` file, e.g. setting field values there,
and helpers to compute JSON Pointers based on the filename+linenumber.
For now you need to maintain that file manually.

