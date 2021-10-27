package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/code-cord/cc.core.server/api"
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
	Policy api.JoinPolicy `json:"policy"`
	Code   string         `json:"code"`
}

// StreamConfigRequest represents stream configuration request model.
type StreamConfigRequest struct {
	PreferredPort int                  `json:"port"`
	PreferredIP   string               `json:"ip"`
	LaunchMode    api.StreamLaunchMode `json:"launch"`
}

// StreamHostInfoRequest represents stream host info request model.
type StreamHostInfoRequest struct {
	Name     string `json:"username"`
	AvatarID string `json:"avatarId"`
}

// StreamOwnerInfoResponse represents stream owner info response model.
type StreamOwnerInfoResponse struct {
	UUID        string               `json:"streamUUID"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	StartedAt   time.Time            `json:"startedAt"`
	JoinPolicy  api.JoinPolicy       `json:"joinPolicy"`
	JoinCode    string               `json:"joinCode,omitempty"`
	Port        int                  `json:"port"`
	IP          string               `json:"ip"`
	LaunchMode  api.StreamLaunchMode `json:"launchMode"`
	HostInfo    HostOwnerInfo        `json:"host"`
	Auth        *AuthorizationInfo   `json:"auth,omitempty"`
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
	UUID        string         `json:"streamUUID"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	JoinPolicy  api.JoinPolicy `json:"joinPolicy"`
	StartedAt   time.Time      `json:"startedAt"`
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
	UUID     string                `json:"uuid"`
	Name     string                `json:"name"`
	AvatarID string                `json:"avatarId"`
	Status   api.ParticipantStatus `json:"status"`
}

// PathcStreamRequest represents patch stream request model.
type PatchStreamRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Join        *JoinPolicyRequest     `json:"join,omitempty"`
	Host        *StreamHostInfoRequest `json:"host,omitempty"`
}

// Rules returns custom validation rules for the request model.
func (req *CreateStreamRequest) Rules() map[string][]string {
	rules := map[string][]string{
		// stream rules.
		"name": {
			"required",
			"max:42",
		},
		"description": {
			"max:96",
		},
		// join policy rules.
		"policy": {
			"required",
			fmt.Sprintf("in:%s", strings.Join([]string{
				string(api.JoinPolicyAuto),
				string(api.JoinPolicyByCode),
				string(api.JoinPolicyHostResolve),
			}, ",")),
		},
		// host info rules.
		"username": {
			"required",
			"max:32",
		},
	}
	if req.Join.Policy == api.JoinPolicyByCode {
		rules["code"] = []string{
			"required",
			"digits:6",
		}
	}

	// stream config rules.
	if req.Stream.LaunchMode != "" {
		rules["launch"] = []string{
			fmt.Sprintf("in:%s", strings.Join([]string{
				string(api.StreamLaunchModeDockerContainer),
				string(api.StreamLaunchModeSingletonApp),
			}, ",")),
		}
	}
	if req.Stream.PreferredIP != "" {
		rules["ip"] = []string{
			"ip",
		}
	}
	if req.Stream.PreferredPort != 0 {
		rules["port"] = []string{
			"min:0",
		}
	}

	return rules
}

// Rules returns custom validation rules for the request model.
func (req *ParticipantJoinRequest) Rules() map[string][]string {
	return map[string][]string{
		"name": {
			"required",
			"max:32",
		},
	}
}

// Rules returns custom validation rules for the request model.
func (req *PatchStreamRequest) Rules() map[string][]string {
	rules := map[string][]string{}
	if req.Name != "" {
		rules["name"] = []string{
			"required",
			"max:32",
		}
	}

	if req.Description != "" {
		rules["description"] = []string{
			"max:96",
		}
	}

	if req.Join != nil {
		rules["policy"] = []string{
			fmt.Sprintf("in:%s", strings.Join([]string{
				string(api.JoinPolicyAuto),
				string(api.JoinPolicyByCode),
				string(api.JoinPolicyHostResolve),
			}, ",")),
		}

		if req.Join.Policy == api.JoinPolicyByCode {
			rules["code"] = []string{
				"required",
				"digits:6",
			}
		}
	}

	if req.Host != nil {
		rules["username"] = []string{
			"required",
			"max:32",
		}
	}

	return rules
}
