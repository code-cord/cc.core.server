package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/code-cord/cc.core.server/service"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type participantInfo struct {
	UUID        string                    `json:"uuid"`
	Name        string                    `json:"name"`
	AvatarID    string                    `json:"avatar,omitempty"`
	IP          string                    `json:"ip"`
	Status      service.ParticipantStatus `json:"status"`
	pendingChan chan bool
}

// JoinParticipant joins a new particiant to the stream.
func (s *Server) JoinParticipant(
	joinCodectx context.Context, streamUUID, joinCode string, p service.Participant) (
	*service.JoinParticipantDecision, error) {
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
		Status:      service.ParticipantStatusPending,
		pendingChan: make(chan bool),
	}
	streamData.pendingParticipants.Store(pInfo.UUID, pInfo)
	defer streamData.pendingParticipants.Delete(pInfo.UUID)

	joinDesicion := new(service.JoinParticipantDecision)
	switch stream.Join.Policy {
	case service.JoinPolicyAuto:
		joinDesicion.JoinAllowed = true
	case service.JoinPolicyByCode:
		if stream.Join.Code != joinCode {
			return nil, errors.New("invalid join code")
		}
		joinDesicion.JoinAllowed = true
	case service.JoinPolicyHostResolve:
		joinDesicion.JoinAllowed = <-pInfo.pendingChan
	default:
		return nil, errors.New("unknown stream join policy")
	}

	if !joinDesicion.JoinAllowed {
		return joinDesicion, nil
	}

	accessToken, err := generateStreamAccessToken(
		streamUUID, pInfo.UUID, false, streamData.rsaKeys.privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %v", err)
	}

	joinDesicion.AccessToken = accessToken
	pInfo.Status = service.ParticipantStatusActive

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

	go s.addNewParticipant(streamUUID, service.StreamParticipant{
		UUID:     pInfo.UUID,
		Name:     pInfo.Name,
		AvatarID: pInfo.AvatarID,
		Status:   pInfo.Status,
		Host:     false,
	})

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
	[]service.Participant, error) {
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

	streamParticipants := make([]service.Participant, len(participants))
	for i := range participants {
		p := &participants[i]

		streamParticipants[i] = service.Participant{
			UUID:     p.UUID,
			Name:     p.Name,
			AvatarID: p.AvatarID,
			IP:       p.IP,
			Status:   p.Status,
		}
	}

	return streamParticipants, nil
}

// PatchParticipant updates participant info.
func (s *Server) PatchParticipant(ctx context.Context, streamUUID, participantUUID string,
	cfg service.PatchParticipantConfig) (*service.Participant, error) {
	_, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}

	participantRV := s.participantStorage.Default().Load(streamUUID)
	if participantRV == nil {
		return nil, errors.New("could not find participants data")
	}

	var participants []participantInfo
	if err := participantRV.Decode(&participants, json.Unmarshal); err != nil {
		return nil, fmt.Errorf("could not decode participants data: %v", err)
	}

	var p *participantInfo
	for i := range participants {
		if participants[i].UUID == participantUUID {
			p = &participants[i]
			break
		}
	}
	if p == nil {
		return nil, fmt.Errorf("could not find participant by UUID %s", participantUUID)
	}

	if cfg.AvatarID != nil {
		p.AvatarID = *cfg.AvatarID
	}
	if cfg.Name != nil {
		p.Name = *cfg.Name
	}

	go s.updateParticipantInfo(streamUUID, service.StreamParticipant{
		UUID:     p.UUID,
		Name:     p.Name,
		AvatarID: p.AvatarID,
		Status:   p.Status,
		Host:     false,
	})

	return &service.Participant{
		UUID:     p.UUID,
		Name:     p.Name,
		AvatarID: p.AvatarID,
		IP:       p.IP,
		Status:   p.Status,
	}, nil
}

func (s *Server) addNewParticipant(streamUUID string, p service.StreamParticipant) {
	stream, ok := s.streams.Load(streamUUID)
	if !ok {
		logrus.Errorf(
			"could not find running stream by UUID %s to add a new participant", streamUUID)
		return
	}

	module := stream.(streamModule)
	if err := module.handler.NewParticipant(p); err != nil {
		logrus.Errorf("could not add participant to the %s stream: %v", streamUUID, err)
	}
}

func (s *Server) updateParticipantInfo(streamUUID string, p service.StreamParticipant) {
	stream, ok := s.streams.Load(streamUUID)
	if !ok {
		logrus.Errorf(
			"could not find running stream by UUID %s to add a new participant", streamUUID)
		return
	}

	module := stream.(streamModule)
	if err := module.handler.ChangeParticipantInfo(p); err != nil {
		logrus.Errorf("could not change participant %s info: %v", p.UUID, err)
	}
}
