package models

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
	Policy string `json:"policy"`
	Code   string `json:"code"`
}

// StreamConfigRequest represents stream configuration request model.
type StreamConfigRequest struct {
	PreferredPort int    `json:"port"`
	PreferredIP   string `json:"ip"`
	LaunchMode    string `json:"launch"`
}

// StreamHostInfoRequest represents stream host info request model.
type StreamHostInfoRequest struct {
	Name string `json:"name"`
}

// StreamInfoResponse represents stream info response model.
type StreamInfoResponse struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	JoinPolicy  string `json:"joinPolicy"`
}
