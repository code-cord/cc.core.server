package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/code-cord/cc.core.server/api"
	"github.com/google/uuid"
)

type participantInfo struct {
	UUID        string                `json:"uuid"`
	Name        string                `json:"name"`
	AvatarID    string                `json:"avatar,omitempty"`
	IP          string                `json:"ip"`
	Status      api.ParticipantStatus `json:"status"`
	pendingChan chan bool
}

// JoinParticipant joins a new particiant to the stream.
func (s *Server) JoinParticipant(
	joinCodectx context.Context, streamUUID, joinCode string, p api.Participant) (
	*api.JoinParticipantDecision, error) {
	streamRV := s.streamStorage.Default().Load(streamUUID)
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok || streamRV == nil {
		return nil, fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}
	streamData := streamValue.(streamModule)

	var stream streamInfo
	if err := streamRV.Decode(&stream, json.Unmarshal); err != nil {
		return nil, fmt.Errorf("could not decode stream data: %v", err)
	}

	pInfo := participantInfo{
		UUID:        uuid.New().String(),
		Name:        p.Name,
		AvatarID:    p.AvatarID,
		IP:          p.IP,
		Status:      api.ParticipantStatusPending,
		pendingChan: make(chan bool),
	}
	streamData.pendingParticipants.Store(pInfo.UUID, pInfo)
	defer streamData.pendingParticipants.Delete(pInfo.UUID)

	joinDesicion := new(api.JoinParticipantDecision)
	switch stream.Join.Policy {
	case api.JoinPolicyAuto:
		joinDesicion.JoinAllowed = true
	case api.JoinPolicyByCode:
		if stream.Join.Code != joinCode {
			return nil, errors.New("invalid join code")
		}
		joinDesicion.JoinAllowed = true
	case api.JoinPolicyHostResolve:
		joinDesicion.JoinAllowed = <-pInfo.pendingChan
	default:
		return nil, errors.New("unknown stream join policy")
	}

	if !joinDesicion.JoinAllowed {
		return joinDesicion, nil
	}

	accessToken, err := generateStreamAccessToken(
		streamUUID, p.UUID, false, streamData.rsaKeys.privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %v", err)
	}

	joinDesicion.AccessToken = accessToken
	pInfo.Status = api.ParticipantStatusActive

	var participants []participantInfo
	if pRV := s.participantStorage.Default().Load(streamUUID); pRV != nil {
		if err := pRV.Decode(&participants, json.Unmarshal); err != nil {
			return nil, fmt.Errorf("could not decode stream participants data: %v", err)
		}
	}
	participants = append(participants, pInfo)
	if err := s.participantStorage.Default().
		Store(streamUUID, participants, json.Marshal); err != nil {
		return nil, fmt.Errorf("could not add participant: %v", err)
	}

	return joinDesicion, nil
}

// DecideParticipantJoin allows or denies participant to join the stream.
func (s *Server) DecideParticipantJoin(
	ctx context.Context, streamUUID, participantUUID string, joinAllowed bool) error {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}
	streamData := streamValue.(streamModule)

	participantValue, ok := streamData.pendingParticipants.Load(participantUUID)
	if !ok {
		return fmt.Errorf("could not find pending participant by UUID %s", participantUUID)
	}
	participantChan := participantValue.(participantInfo).pendingChan
	participantChan <- joinAllowed

	return nil
}

// StreamParticipants returns list of stream participants.
func (s *Server) StreamParticipants(ctx context.Context, streamUUID string) (
	[]api.Participant, error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}
	streamData := streamValue.(streamModule)

	var participants []participantInfo
	streamData.pendingParticipants.Range(func(key, value interface{}) bool {
		participants = append(participants, value.(participantInfo))

		return true
	})

	if participantRV := s.participantStorage.Default().Load(streamUUID); participantRV != nil {
		var storageParticipants []participantInfo
		if err := participantRV.Decode(&storageParticipants, json.Unmarshal); err != nil {
			return nil, fmt.Errorf("could not decode participants data: %v", err)
		}

		participants = append(participants, storageParticipants...)
	}

	streamParticipants := make([]api.Participant, len(participants))
	for i := range participants {
		p := &participants[i]

		streamParticipants[i] = api.Participant{
			UUID:     p.UUID,
			Name:     p.Name,
			AvatarID: p.AvatarID,
			IP:       p.IP,
			Status:   p.Status,
		}
	}

	return streamParticipants, nil
}
