package api

import "context"

// Server describes server API.
type Server interface {
	Info() ServerInfo
	Ping(ctx context.Context) error
	NewStream(ctx context.Context, cfg StreamConfig) (*StreamOwnerInfo, error)
}

// ServerInfo represents server info model.
type ServerInfo struct {
	Name        string
	Description string
	Version     string
	Meta        map[string]interface{}
}

// StreamConfig represents stream configuration model.
type StreamConfig struct {
	Name        string
	Description string
	Join        StreamJoinPolicyConfig
	Launch      StreamLaunchConfig
	Host        StreamHostConfig
}

// StreamJoinPolicyConfig represents stream join policy configuration model.
type StreamJoinPolicyConfig struct {
	JoinPolicy JoinPolicy
	JoinCode   string
}

// StreamLaunchConfig represents stream launch configuration model.
type StreamLaunchConfig struct {
	PreferredPort int
	PreferredIP   string
	Mode          StreamLaunchMode
}

// StreamHostConfig represents stream host configuration model.
type StreamHostConfig struct {
	Username string
	AvatarID string
	IP       string
}

// StreamOwnerInfo represents stream owner info.
type StreamOwnerInfo struct {
	UUID        string
	Name        string
	Description string
	JoinPolicy  JoinPolicy
	JoinCode    string
	Port        int
	IP          string
	LaunchMode  StreamLaunchMode
	Host        HostInfo
}

// HostInfo represents host of the stream info.
type HostInfo struct {
	UUID     string
	Username string
	AvatarID string
	IP       string
}
