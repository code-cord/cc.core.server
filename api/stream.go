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

// Stream represents stream API.
type Stream interface {
	Start(ctx context.Context) error
	//Info(ctx context.Context)
	Stop(ctx context.Context) error
}

// JoinPolicy represents join to stream policy.
type JoinPolicy string

// StreamLaunchMode represents stream launch mode.
type StreamLaunchMode string
