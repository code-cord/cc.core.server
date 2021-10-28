package api

import "context"

// Stream join policy.
const (
	JoinPolicyAuto        JoinPolicy = "auto"
	JoinPolicyByCode      JoinPolicy = "by_code"
	JoinPolicyHostResolve JoinPolicy = "host_resolve"
)

// Stream launch mode.
const (
	StreamLaunchModeSingletonApp    StreamLaunchMode = "singleton_app"
	StreamLaunchModeDockerContainer StreamLaunchMode = "docker_container"
)

// Stream status.
const (
	StreamStatusRunning  StreamStatus = "running"
	StreamStatusFinished StreamStatus = "finished"
)

// Stream represents stream API.
type Stream interface {
	Start(ctx context.Context) (*StartStreamInfo, error)
	Stop(ctx context.Context) error
	InterruptNotification() <-chan error
}

// StartStreamInfo represents start stream info model.
type StartStreamInfo struct {
	IP   string
	Port int
}

// JoinPolicy represents join to stream policy.
type JoinPolicy string

// StreamLaunchMode represents stream launch mode.
type StreamLaunchMode string

// StreamStatus represents stream status.
type StreamStatus string
