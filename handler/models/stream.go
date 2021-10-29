package models

import (
	"regexp"
	"time"

	"github.com/code-cord/cc.core.server/service"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// CreateStreamRequest represents create stream request model.
type CreateStreamRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Join        JoinPolicyRequest     `json:"join"`
	Stream      StreamConfigRequest   `json:"stream"`
	Host        StreamHostInfoRequest `json:"host"`
}

// JoinPolicyRequest represents join policy request model.
type JoinPolicyRequest struct {
	Policy service.JoinPolicy `json:"policy"`
	Code   string             `json:"code"`
}

// StreamConfigRequest represents stream configuration request model.
type StreamConfigRequest struct {
	PreferredPort int                      `json:"port"`
	PreferredIP   string                   `json:"ip"`
	LaunchMode    service.StreamLaunchMode `json:"launch"`
}

// StreamHostInfoRequest represents stream host info request model.
type StreamHostInfoRequest struct {
	Name     string `json:"username"`
	AvatarID string `json:"avatarId"`
}

// StreamOwnerInfoResponse represents stream owner info response model.
type StreamOwnerInfoResponse struct {
	UUID        string                   `json:"streamUUID"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	StartedAt   time.Time                `json:"startedAt"`
	JoinPolicy  service.JoinPolicy       `json:"joinPolicy"`
	JoinCode    string                   `json:"joinCode,omitempty"`
	Port        int                      `json:"port"`
	IP          string                   `json:"ip"`
	LaunchMode  service.StreamLaunchMode `json:"launchMode"`
	HostInfo    HostOwnerInfo            `json:"host"`
	Auth        *AuthorizationInfo       `json:"auth,omitempty"`
}

// HostOwnerInfo represents host owner info response.
type HostOwnerInfo struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	AvatarID string `json:"avatarId"`
	IP       string `json:"ip"`
}

// AuthorizationInfo represents authorization info model.
type AuthorizationInfo struct {
	AccessToken string `json:"accessToken"`
	Type        string `json:"type"`
}

// StreamPublicInfoResponse represents stream public info response model.
type StreamPublicInfoResponse struct {
	UUID        string             `json:"streamUUID"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	JoinPolicy  service.JoinPolicy `json:"joinPolicy"`
	StartedAt   time.Time          `json:"startedAt"`
	FinishedAt  *time.Time         `json:"finishedAt,omitempty"`
}

// ParticipantJoinRequest represents participant join request model.
type ParticipantJoinRequest struct {
	Name     string `json:"name"`
	AvatarID string `json:"avatarId"`
	JoinCode string `json:"joinCode,omitempty"`
}

// ParticipantJoinResponse represents participant join response model.
type ParticipantJoinResponse struct {
	Allowed     bool   `json:"allowed"`
	AccessToken string `json:"accessToken,omitempty"`
}

// ParticipantResponse represents participant response model.
type ParticipantResponse struct {
	UUID     string                    `json:"uuid"`
	Name     string                    `json:"name"`
	AvatarID string                    `json:"avatarId"`
	Status   service.ParticipantStatus `json:"status"`
}

// PathcStreamRequest represents patch stream request model.
type PatchStreamRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Join        *JoinPolicyRequest     `json:"join,omitempty"`
	Host        *StreamHostInfoRequest `json:"host,omitempty"`
}

// Validate validates request model.
func (req *CreateStreamRequest) Validate() error {
	errs := validation.Errors{
		"name": validation.Validate(req.Name,
			validation.Required,
			validation.Length(5, 32),
		),
		"description": validation.Validate(req.Description,
			validation.Length(0, 96),
		),
		"join.policy": validation.Validate(req.Join.Policy,
			validation.Required,
			validation.In(
				service.JoinPolicyAuto,
				service.JoinPolicyByCode,
				service.JoinPolicyHostResolve,
			),
		),
		"host.username": validation.Validate(req.Host.Name,
			validation.Required,
			validation.Length(5, 32),
		),
	}

	if req.Join.Policy == service.JoinPolicyByCode {
		errs["join.code"] = validation.Validate(req.Join.Code,
			validation.Required,
			validation.Match(regexp.MustCompile("^[0-9]{6}$")),
		)
	}

	if req.Stream.LaunchMode != "" {
		errs["stream.launch"] = validation.Validate(req.Stream.LaunchMode,
			validation.In(
				service.StreamLaunchModeDockerContainer,
				service.StreamLaunchModeStandaloneApp,
			),
		)
	}

	if req.Stream.PreferredIP != "" {
		errs["stream.ip"] = validation.Validate(req.Stream.PreferredIP,
			is.IP,
		)
	}

	if req.Stream.PreferredPort != 0 {
		errs["stream.port"] = validation.Validate(req.Stream.PreferredPort,
			validation.Min(0),
		)
	}

	return errs.Filter()
}

// Validate validates request model.
func (req *ParticipantJoinRequest) Validate() error {
	return validation.Errors{
		"name": validation.Validate(req.Name,
			validation.Required,
			validation.Length(5, 32),
		),
	}.Filter()
}

// Validate validates request model.
func (req *PatchStreamRequest) Validate() error {
	errs := validation.Errors{}

	if req.Name != nil {
		errs["name"] = validation.Validate(req.Name,
			validation.Length(5, 32),
		)
	}

	if req.Description != nil {
		errs["description"] = validation.Validate(req.Description,
			validation.Length(0, 96),
		)
	}

	if req.Join != nil {
		errs["join.policy"] = validation.Validate(req.Join.Policy,
			validation.In(
				service.JoinPolicyAuto,
				service.JoinPolicyByCode,
				service.JoinPolicyHostResolve,
			),
		)

		if req.Join.Policy == service.JoinPolicyByCode {
			errs["join.code"] = validation.Validate(req.Join.Code,
				validation.Required,
				validation.Match(regexp.MustCompile("^[0-9]{6}$")),
			)
		}
	}

	if req.Host != nil {
		errs["host.username"] = validation.Validate(req.Join.Code,
			validation.Length(5, 32),
		)
	}

	return errs.Filter()
}
