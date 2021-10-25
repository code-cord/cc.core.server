package models

import (
	"fmt"
	"strings"

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
	Name     string `json:"userName"`
	AvatarID string `json:"avatarId"`
}

// StreamOwnerInfoResponse represents stream owner info response model.
type StreamOwnerInfoResponse struct {
	UUID        string               `json:"streamUUID"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	JoinPolicy  api.JoinPolicy       `json:"joinPolicy"`
	JoinCode    string               `json:"joinCode"`
	Port        int                  `json:"port"`
	IP          string               `json:"ip"`
	LaunchMode  api.StreamLaunchMode `json:"launchMode"`
	HostInfo    HostOwnerInfo        `json:"host"`
}

// HostOwnerInfo represents host owner info response.
type HostOwnerInfo struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	AvatarID string `json:"avatarId"`
	IP       string `json:"ip"`
}

// StreamInfoResponse represents stream info response model.
type StreamInfoResponse struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	JoinPolicy  string `json:"joinPolicy"`
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
		"userName": {
			"required",
			"max:32",
		},
	}
	if req.Join.Policy == api.JoinPolicyByCode {
		rules["code"] = []string{
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
