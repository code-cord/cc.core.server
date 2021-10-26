package api

import (
	"context"
	"time"
)

// Participant status.
const (
	ParticipantStatusActive  ParticipantStatus = "active"
	ParticipantStatusBlocked ParticipantStatus = "blocked"
	ParticipantStatusPending ParticipantStatus = "pending"
)

// Server describes server API.
type Server interface {
	Info() ServerInfo
	Ping(ctx context.Context) error
	NewStream(ctx context.Context, cfg StreamConfig) (*StreamOwnerInfo, error)
	StreamInfo(ctx context.Context, streamUUID string) *StreamPublicInfo
	JoinParticipant(ctx context.Context, streamUUID, joinCode string, p Participant) (
		*JoinParticipantDecision, error)
	DecideParticipantJoin(
		ctx context.Context, streamUUID, participantUUID string, joinAllowed bool) error
	StreamParticipants(ctx context.Context, streamUUID string) ([]Participant, error)
	FinishStream(ctx context.Context, streamUUID string) error
	NewStreamHostToken(ctx context.Context, streamUUID, subject string) (*StreamAuthInfo, error)
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
	Subject     string
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
	StartedAt   time.Time
	JoinPolicy  JoinPolicy
	JoinCode    string
	Port        int
	IP          string
	LaunchMode  StreamLaunchMode
	Host        HostInfo
	Auth        StreamAuthInfo
}

// HostInfo represents host of the stream info.
type HostInfo struct {
	UUID     string
	Username string
	AvatarID string
	IP       string
}

// StreamAuthInfo represents stream authentication info model.
type StreamAuthInfo struct {
	AccessToken string
	Type        string
}

// StreamPublicInfo represents stream public info model.
type StreamPublicInfo struct {
	UUID        string
	Name        string
	Description string
	JoinPolicy  JoinPolicy
	StartedAt   time.Time
}

// Participant represents participant model.
type Participant struct {
	UUID     string
	Name     string
	AvatarID string
	IP       string
	Status   ParticipantStatus
}

// ParticipantStatus represents participant status type.
type ParticipantStatus string

// JoinParticipantDecision represents join participant decision model.
type JoinParticipantDecision struct {
	JoinAllowed bool
	AccessToken string
}
