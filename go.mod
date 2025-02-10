module github.com/o3willard-AI/SSSonector

go 1.21

require (
	github.com/gosnmp/gosnmp v1.37.0
	github.com/seccomp/libseccomp-golang v0.10.0
	github.com/stretchr/testify v1.8.4
	github.com/vishvananda/netlink v1.3.0
	go.uber.org/zap v1.27.0
	golang.org/x/sys v0.13.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/soniah/gosnmp => github.com/gosnmp/gosnmp v1.37.0
