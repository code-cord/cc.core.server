package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"sync"
	"time"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/stream"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewStream starts a new stream.
func (s *Server) NewStream(ctx context.Context, cfg api.StreamConfig) (
	*api.StreamOwnerInfo, error) {
	// generate stream access keys.
	keys, err := generateRSAKeys()
	if err != nil {
		return nil, fmt.Errorf("could not generate stream access keys: %v", err)
	}

	if cfg.Launch.Mode == "" {
		cfg.Launch.Mode = api.StreamLaunchModeSingletonApp
	}

	var streamHandler api.Stream
	streamUUID := uuid.New().String()
	hostUUID := uuid.New().String()
	switch cfg.Launch.Mode {
	case api.StreamLaunchModeSingletonApp:
		streamHandler = stream.NewStandaloneStream(stream.StandaloneStreamConfig{
			PreferedIP:   cfg.Launch.PreferredIP,
			PreferedPort: cfg.Launch.PreferredPort,
			BinPath:      s.opts.BinFolder,
		})
	case api.StreamLaunchModeDockerContainer:
		streamHandler = stream.NewDockerContainerStream(stream.DockerContainerStreamConfig{
			StreamUUID:      streamUUID,
			ContainerPrefix: s.opts.StreamContainerPrefix,
			DockerImage:     s.opts.StreamImage,
			PreferedPort:    cfg.Launch.PreferredPort,
			PreferedIP:      cfg.Launch.PreferredIP,
		})
	default:
		return nil, fmt.Errorf("invalid launch mode: %v", cfg.Launch.Mode)
	}

	// generate host access token.
	token, err := generateStreamAccessToken(streamUUID, hostUUID, true, keys.privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not authorize host user for the stream: %v", err)
	}

	streamLaunchInfo, err := streamHandler.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not run stream instance: %v", err)
	}

	var startStreamErr error
	defer func() {
		if startStreamErr != nil {
			if err := streamHandler.Stop(ctx); err != nil {
				logrus.Errorf("could not stop container: %v", err)
			}
		}
	}()

	tcpAddress := fmt.Sprintf("%s:%d", streamLaunchInfo.IP, streamLaunchInfo.Port)
	rcpClient, err := connectToStream(tcpAddress, defaultConnectToStreamRetryCount)
	if err != nil {
		startStreamErr = fmt.Errorf("could not connect to the running stream: %v", err)
		return nil, startStreamErr
	}

	// listening stream interrupt event.
	go s.listenStreamInterruptEvent(streamUUID, streamHandler.InterruptNotification())

	newStreamInfo := streamInfo{
		Stream:      streamHandler,
		startedAt:   time.Now().UTC(),
		name:        cfg.Name,
		description: cfg.Description,
		ip:          streamLaunchInfo.IP,
		port:        streamLaunchInfo.Port,
		join:        cfg.Join,
		rcpClient:   rcpClient,
		hostInfo: api.HostInfo{
			UUID:     hostUUID,
			Username: cfg.Host.Username,
			AvatarID: cfg.Host.AvatarID,
			IP:       cfg.Host.IP,
		},
		launchMode:             cfg.Launch.Mode,
		participants:           new(sync.Map),
		pendingParticipantsMap: new(sync.Map),
		rsaKeys:                keys,
		subject:                cfg.Subject,
	}
	s.streams.Store(streamUUID, newStreamInfo)

	return buildStreamOwnerInfo(&newStreamInfo, streamUUID, token), nil
}

// StreamInfo returns public stream info by stream UUID.
func (s *Server) StreamInfo(ctx context.Context, streamUUID string) *api.StreamPublicInfo {
	stream, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil
	}

	info := stream.(streamInfo)

	return &api.StreamPublicInfo{
		UUID:        streamUUID,
		Name:        info.name,
		Description: info.description,
		JoinPolicy:  info.join.JoinPolicy,
		StartedAt:   info.startedAt,
	}
}

// FinishStream finishes running stream.
func (s *Server) FinishStream(ctx context.Context, streamUUID string) error {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}
	streamData := streamValue.(streamInfo)

	if err := streamData.rcpClient.Close(); err != nil {
		return fmt.Errorf("could not close connection to the %s stream: %v", streamUUID, err)
	}

	if err := streamData.Stream.Stop(ctx); err != nil {
		return err
	}

	s.streams.Delete(streamUUID)

	return nil
}

// StreamKey returns stream public key info.
func (s *Server) StreamKey(ctx context.Context, streamUUID string) (*rsa.PublicKey, error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}
	streamData := streamValue.(streamInfo)

	return streamData.rsaKeys.publicKey, nil
}

// PatchStream updates stream info.
func (s *Server) PatchStream(ctx context.Context, streamUUID string, cfg api.PatchStreamConfig) (
	*api.StreamOwnerInfo, error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}
	streamData := streamValue.(streamInfo)

	if cfg.Name != nil {
		streamData.name = *cfg.Name
	}

	if cfg.Description != nil {
		streamData.description = *cfg.Description
	}

	if cfg.Join != nil {
		streamData.join = api.StreamJoinPolicyConfig{
			JoinPolicy: cfg.Join.JoinPolicy,
			JoinCode:   cfg.Join.JoinCode,
		}
	}

	if cfg.Host != nil {
		streamData.hostInfo.Username = cfg.Host.Username
		streamData.hostInfo.AvatarID = cfg.Host.AvatarID
	}

	s.streams.Store(streamUUID, streamData)

	return buildStreamOwnerInfo(&streamData, streamUUID, ""), nil
}
