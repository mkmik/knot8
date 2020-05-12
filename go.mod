module knot8.io

go 1.14

require (
	github.com/alecthomas/kong v0.2.7
	github.com/hashicorp/go-getter v1.4.1
	github.com/mattn/go-isatty v0.0.12
	github.com/mkmik/multierror v0.3.0
	github.com/vmware-labs/yaml-jsonpointer v0.1.0
	github.com/vmware-labs/yaml-jsonpointer/yamled v0.1.0
	golang.org/x/text v0.3.2
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

replace gopkg.in/yaml.v3 => github.com/atomatt/yaml v0.0.0-20200228174225-55c5cf55e3ee
