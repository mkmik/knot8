module knot8.io

go 1.14

require (
	github.com/alecthomas/kong v0.3.0
	github.com/go-openapi/jsonpointer v0.19.5
	github.com/google/go-jsonnet v0.18.0
	github.com/hashicorp/go-getter v1.5.11
	github.com/mattn/go-isatty v0.0.14
	github.com/mkmik/multierror v0.3.0
	github.com/pelletier/go-toml v1.9.4
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/vmware-labs/go-yaml-edit v0.3.0
	github.com/vmware-labs/yaml-jsonpointer v0.1.1
	golang.org/x/text v0.3.7
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace gopkg.in/yaml.v3 => github.com/atomatt/yaml v0.0.0-20200228174225-55c5cf55e3ee
