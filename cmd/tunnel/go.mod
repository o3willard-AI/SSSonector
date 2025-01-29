module github.com/o3willard-AI/SSSonector/cmd/tunnel

go 1.21

require (
	github.com/o3willard-AI/SSSonector/internal/adapter v0.0.0
	github.com/o3willard-AI/SSSonector/internal/config v0.0.0
	github.com/o3willard-AI/SSSonector/internal/tunnel v0.0.0
	go.uber.org/zap v1.27.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
)

replace (
	github.com/o3willard-AI/SSSonector/internal/adapter => ../../internal/adapter
	github.com/o3willard-AI/SSSonector/internal/cert => ../../internal/cert
	github.com/o3willard-AI/SSSonector/internal/config => ../../internal/config
	github.com/o3willard-AI/SSSonector/internal/throttle => ../../internal/throttle
	github.com/o3willard-AI/SSSonector/internal/tunnel => ../../internal/tunnel
)
