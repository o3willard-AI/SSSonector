module github.com/o3willard-AI/SSSonector

go 1.21

require (
	github.com/gosnmp/gosnmp v1.38.0
	go.uber.org/zap v1.27.0
	golang.org/x/time v0.10.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/soniah/gosnmp => github.com/gosnmp/gosnmp v1.37.0
