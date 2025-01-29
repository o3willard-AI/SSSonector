module github.com/o3willard-AI/SSSonector/internal/tunnel

go 1.21

require (
	github.com/o3willard-AI/SSSonector/internal/adapter v0.0.0
	github.com/o3willard-AI/SSSonector/internal/cert v0.0.0
	github.com/o3willard-AI/SSSonector/internal/config v0.0.0
	github.com/o3willard-AI/SSSonector/internal/throttle v0.0.0
	go.uber.org/zap v1.27.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
)

replace (
	github.com/o3willard-AI/SSSonector/internal/adapter => ../adapter
	github.com/o3willard-AI/SSSonector/internal/cert => ../cert
	github.com/o3willard-AI/SSSonector/internal/config => ../config
	github.com/o3willard-AI/SSSonector/internal/throttle => ../throttle
)
