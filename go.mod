module knot8.io

go 1.14

require (
	github.com/alecthomas/kong v0.2.11
	github.com/go-openapi/jsonpointer v0.19.3
	github.com/google/go-jsonnet v0.16.0
	github.com/hashicorp/go-getter v1.4.1
	github.com/mattn/go-isatty v0.0.12
	github.com/mkmik/multierror v0.3.0
	github.com/pelletier/go-toml v1.8.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vmware-labs/go-yaml-edit v0.3.0
	github.com/vmware-labs/yaml-jsonpointer v0.1.1
	golang.org/x/text v0.3.3
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

replace gopkg.in/yaml.v3 => github.com/atomatt/yaml v0.0.0-20200228174225-55c5cf55e3ee
