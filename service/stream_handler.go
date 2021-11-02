package service

// StreamHandler represents stream handler API.
type StreamHandler interface {
	NewParticipant(p StreamParticipant) error
	ChangeParticipantInfo(p StreamParticipant) error
}

// StreamParticipant represents stream participant model.
type StreamParticipant struct {
	UUID     string            `json:"uuid"`
	Name     string            `json:"name"`
	AvatarID string            `json:"avatarId,omitempty"`
	Status   ParticipantStatus `json:"status"`
	Host     bool              `json:"isHost,omitempty"`
}
