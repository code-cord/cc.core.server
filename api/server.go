package api

import (
	"context"
	"crypto/rsa"
	"io"
	"time"

	"github.com/golang-jwt/jwt"
)

// Participant status.
const (
	ParticipantStatusActive  ParticipantStatus = "active"
	ParticipantStatusBlocked ParticipantStatus = "blocked"
	ParticipantStatusPending ParticipantStatus = "pending"
)

// Stream sort field.
const (
	StreamSortByFieldUUID       StreamSortByField = "uuid"
	StreamSortByFieldName       StreamSortByField = "name"
	StreamSortByFieldLaunchMode StreamSortByField = "mode"
	StreamSortByFieldStarted    StreamSortByField = "started"
	StreamSortByFieldStatus     StreamSortByField = "status"
)

// Stream sort order.
const (
	StreamSortOrderDesc StreamSortOrder = "desc"
	StreamSortOrderAsc  StreamSortOrder = "asc"
)

// Server storage.
const (
	ServerStorageAvatar      ServerStorage = "avatar"
	ServerStorageParticipant ServerStorage = "participant"
	ServerStorageStream      ServerStorage = "stream"
)

// Server describes server API.
type Server interface {
	Info() ServerInfo
	Ping(ctx context.Context) error
	NewStream(ctx context.Context, cfg StreamConfig) (*StreamOwnerInfo, error)
	StreamInfo(ctx context.Context, streamUUID string) (*StreamPublicInfo, error)
	JoinParticipant(ctx context.Context, streamUUID, joinCode string, p Participant) (
		*JoinParticipantDecision, error)
	DecideParticipantJoin(
		ctx context.Context, streamUUID, participantUUID string, joinAllowed bool) error
	StreamParticipants(ctx context.Context, streamUUID string) ([]Participant, error)
	FinishStream(ctx context.Context, streamUUID string) error
	NewStreamHostToken(ctx context.Context, streamUUID, subject string) (*AuthInfo, error)
	NewServerToken(ctx context.Context, claims *jwt.StandardClaims) (*AuthInfo, error)
	StreamKey(ctx context.Context, streamUUID string) (*rsa.PublicKey, error)
	PatchStream(ctx context.Context, streamUUID string, cfg PatchStreamConfig) (
		*StreamOwnerInfo, error)
	NewAvatar(ctx context.Context, contentType string, r io.Reader) (string, error)
	AvatarRestrictions() AvatarRestrictions
	AvatarByID(ctx context.Context, avatarID string) ([]byte, string, error)
	StreamList(ctx context.Context, filter StreamFilter) (*StreamList, error)
	StorageBackup(ctx context.Context, storageName ServerStorage, w io.Writer) error
}

// AvatarRestrictions represents avatar restrictions model.
type AvatarRestrictions struct {
	MaxFileSize int64
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
	Auth        *AuthInfo
}

// HostInfo represents host of the stream info.
type HostInfo struct {
	UUID     string
	Username string
	AvatarID string
	IP       string
}

// AuthInfo represents authentication info model.
type AuthInfo struct {
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
	FinishedAt  *time.Time
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

// PatchStreamConfig represents patch stream configuration model.
type PatchStreamConfig struct {
	Name        *string
	Description *string
	Join        *StreamJoinPolicyConfig
	Host        *StreamHostConfig
}

// StreamFilter represents stream filter model.
type StreamFilter struct {
	SearchPhrase string
	LaunchModes  []StreamLaunchMode
	Statuses     []StreamStatus
	SortBy       StreamSortByField
	SortOrder    StreamSortOrder
	PageSize     int
	Page         int
}

// StreamSortByField represents sort by field type.
type StreamSortByField string

// StreamSortOrder represents stream sort order.
type StreamSortOrder string

// StreamList represents paginated streams model.
type StreamList struct {
	Streams  []StreamInfo
	Page     int
	PageSize int
	Count    int
	HasNext  bool
	Total    int
}

// StreamInfo represents stream info model.
type StreamInfo struct {
	UUID        string
	Name        string
	Description string
	IP          string
	Port        int
	LaunchMode  StreamLaunchMode
	StartedAt   time.Time
	FinishedAt  *time.Time
	Status      StreamStatus
	Join        StreamJoinPolicyConfig
	Host        HostInfo
}

// ServerStorage represents server storage type.
type ServerStorage string
