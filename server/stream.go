package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"sync"
	"time"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/stream"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultConnectToStreamRetryCount   = 3
	defaultConnectToStreamRetryTimeout = 2 * time.Second
	defaultStreamTokenType             = "bearer"
)

type streamModule struct {
	api.Stream
	rpcClient           *rpc.Client
	pendingParticipants *sync.Map
	rsaKeys             *rsaKeys
}

type streamInfo struct {
	UUID        string               `json:"uuid"`
	Name        string               `json:"name"`
	Description string               `json:"desc,omitempty"`
	IP          string               `json:"ip"`
	Port        int                  `json:"port"`
	LaunchMode  api.StreamLaunchMode `json:"mode"`
	StartedAt   time.Time            `json:"startedAt"`
	FinishedAt  *time.Time           `json:"finishedAt,omitempty"`
	Subject     string               `json:"sub,omitempty"`
	Status      api.StreamStatus     `json:"status"`
	Join        streamJoinInfo       `json:"join"`
	Host        streamHostInfo       `json:"host"`
}

type streamJoinInfo struct {
	Code   string         `json:"code,omitempty"`
	Policy api.JoinPolicy `json:"policy"`
}

type streamHostInfo struct {
	UUID     string `json:"uuid"`
	Username string `json:"name"`
	AvatarID string `json:"avatar,omitempty"`
	IP       string `json:"ip"`
}

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

	streamUUID := uuid.New().String()
	hostUUID := uuid.New().String()
	streamHandler, err := s.newStreamHandler(cfg, streamUUID)
	if err != nil {
		return nil, err
	}

	// generate host access token.
	token, err := generateStreamAccessToken(streamUUID, hostUUID, true, keys.privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not authorize host user for the stream: %v", err)
	}

	// start stream and connect.
	rpcClient, startInfo, err := startStreamAndConnect(ctx, streamHandler)
	if err != nil {
		return nil, err
	}

	// listening stream interrupt event.
	go s.listenStreamInterruptEvent(streamUUID, streamHandler.InterruptNotification())

	// store stream data.
	info := streamInfo{
		UUID:        streamUUID,
		Name:        cfg.Name,
		Description: cfg.Description,
		IP:          startInfo.IP,
		Port:        startInfo.Port,
		LaunchMode:  cfg.Launch.Mode,
		StartedAt:   time.Now().UTC(),
		Subject:     cfg.Subject,
		Status:      api.StreamStatusRunning,
		Join: streamJoinInfo{
			Code:   cfg.Join.JoinCode,
			Policy: cfg.Join.JoinPolicy,
		},
		Host: streamHostInfo{
			UUID:     hostUUID,
			Username: cfg.Host.Username,
			AvatarID: cfg.Host.AvatarID,
			IP:       cfg.Host.IP,
		},
	}
	if err := s.streamStorage.Default().Store(streamUUID, info, json.Marshal); err != nil {
		s.killStream(ctx, streamUUID)
		return nil, fmt.Errorf("could not store %s stream data: %v", streamUUID, err)
	}

	s.streams.Store(streamUUID, streamModule{
		rpcClient:           rpcClient,
		pendingParticipants: new(sync.Map),
		rsaKeys:             keys,
		Stream:              streamHandler,
	})

	return buildStreamOwnerInfo(&info, token), nil
}

// StreamInfo returns public stream info by stream UUID.
func (s *Server) StreamInfo(ctx context.Context, streamUUID string) (*api.StreamPublicInfo, error) {
	streamRV := s.streamStorage.Default().Load(streamUUID)
	if streamRV == nil {
		return nil, os.ErrNotExist
	}

	var info streamInfo
	if err := streamRV.Decode(&info, json.Unmarshal); err != nil {
		return nil, fmt.Errorf("could not decode stream data: %v", err)
	}

	return &api.StreamPublicInfo{
		UUID:        streamUUID,
		Name:        info.Name,
		Description: info.Description,
		JoinPolicy:  info.Join.Policy,
		StartedAt:   info.StartedAt,
		FinishedAt:  info.FinishedAt,
	}, nil
}

// FinishStream finishes running stream.
func (s *Server) FinishStream(ctx context.Context, streamUUID string) error {
	_, ok := s.streams.Load(streamUUID)
	if !ok {
		return fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}

	s.killStream(ctx, streamUUID)

	return nil
}

// StreamKey returns stream public key info.
func (s *Server) StreamKey(ctx context.Context, streamUUID string) (*rsa.PublicKey, error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}
	streamData := streamValue.(streamModule)

	return streamData.rsaKeys.publicKey, nil
}

// PatchStream updates stream info.
func (s *Server) PatchStream(ctx context.Context, streamUUID string, cfg api.PatchStreamConfig) (
	*api.StreamOwnerInfo, error) {
	streamRV := s.streamStorage.Default().Load(streamUUID)
	_, ok := s.streams.Load(streamUUID)
	if !ok || streamRV == nil {
		return nil, fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}

	var info streamInfo
	if err := streamRV.Decode(&info, json.Unmarshal); err != nil {
		return nil, fmt.Errorf("could not decode stream data: %v", err)
	}

	if cfg.Name != nil {
		info.Name = *cfg.Name
	}

	if cfg.Description != nil {
		info.Description = *cfg.Description
	}

	if cfg.Join != nil {
		info.Join = streamJoinInfo{
			Code:   cfg.Join.JoinCode,
			Policy: cfg.Join.JoinPolicy,
		}
	}

	if cfg.Host != nil {
		info.Host.Username = cfg.Host.Username
		info.Host.AvatarID = cfg.Host.AvatarID
	}

	if err := s.streamStorage.Default().Store(streamUUID, info, json.Marshal); err != nil {
		return nil, fmt.Errorf("could not update stream info: %v", err)
	}

	return buildStreamOwnerInfo(&info, ""), nil
}

// NewStreamHostToken generates new access token for the host of the stream.
func (s *Server) NewStreamHostToken(ctx context.Context, streamUUID, subject string) (
	*api.StreamAuthInfo, error) {
	streamRV := s.streamStorage.Default().Load(streamUUID)
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok || streamRV == nil {
		return nil, fmt.Errorf("could not find running stream by UUID %s", streamUUID)
	}
	streamData := streamValue.(streamModule)

	var info streamInfo
	if err := streamRV.Decode(&info, json.Unmarshal); err != nil {
		return nil, fmt.Errorf("could not decode stream data: %v", err)
	}

	if info.Subject == "" || info.Subject != subject {
		return nil, errors.New("could not verify stream subject")
	}

	token, err := generateStreamAccessToken(
		streamUUID, info.Host.UUID, true, streamData.rsaKeys.privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %v", err)
	}

	return &api.StreamAuthInfo{
		AccessToken: token,
		Type:        defaultStreamTokenType,
	}, nil
}

func (s *Server) newStreamHandler(cfg api.StreamConfig, streamUUID string) (api.Stream, error) {
	switch cfg.Launch.Mode {
	case api.StreamLaunchModeSingletonApp:
		return stream.NewStandaloneStream(stream.StandaloneStreamConfig{
			PreferedIP:   cfg.Launch.PreferredIP,
			PreferedPort: cfg.Launch.PreferredPort,
			BinPath:      s.opts.BinFolder,
		}), nil
	case api.StreamLaunchModeDockerContainer:
		return stream.NewDockerContainerStream(stream.DockerContainerStreamConfig{
			StreamUUID:      streamUUID,
			ContainerPrefix: s.opts.StreamContainerPrefix,
			DockerImage:     s.opts.StreamImage,
			PreferedPort:    cfg.Launch.PreferredPort,
			PreferedIP:      cfg.Launch.PreferredIP,
		}), nil
	}

	return nil, fmt.Errorf("invalid launch mode: %v", cfg.Launch.Mode)
}

func (s *Server) listenStreamInterruptEvent(streamUUID string, intChan <-chan error) {
	err := <-intChan
	if err != nil {
		logrus.Errorf("stream %s has been interrupted: %v", streamUUID, err)
	}

	s.streams.Delete(streamUUID)
	s.killStream(context.Background(), streamUUID)
}

func (s *Server) killStream(ctx context.Context, streamUUID string) {
	if streamValue, ok := s.streams.Load(streamUUID); ok {
		stream := streamValue.(streamModule)
		if err := stream.rpcClient.Close(); err != nil {
			logrus.Errorf("could not close %s stream connection: %v", streamUUID, err)
		}

		if err := stream.Stop(ctx); err != nil {
			logrus.Errorf("could not stop %s stream: %v", streamUUID, err)
		}

		s.streams.Delete(streamUUID)
	}

	streamRV := s.streamStorage.Default().Load(streamUUID)
	if streamRV == nil {
		return
	}

	var stream streamInfo
	if err := streamRV.Decode(&stream, json.Unmarshal); err != nil {
		logrus.Errorf("could not load stream %s data to finish: %v", streamUUID, err)
		return
	}

	now := time.Now().UTC()
	stream.FinishedAt = &now
	stream.Status = api.StreamStatusFinished
	if err := s.streamStorage.Default().Store(streamUUID, stream, json.Marshal); err != nil {
		logrus.Errorf("could not store %s stream data to finish: %v", streamUUID, err)
	}
}

func generateStreamAccessToken(
	streamUUID, participantUUID string, isHost bool, privateKey *rsa.PrivateKey) (string, error) {
	claims := &jwt.MapClaims{
		"streamUUID": streamUUID,
		"UUID":       participantUUID,
		"host":       isHost,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func generateRSAKeys() (*rsaKeys, error) {
	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not generate private RSA key: %v", err)
	}

	return &rsaKeys{
		privateKey: privatekey,
		publicKey:  &privatekey.PublicKey,
	}, nil
}

func startStreamAndConnect(ctx context.Context, stream api.Stream) (
	*rpc.Client, *api.StartStreamInfo, error) {
	streamLaunchInfo, err := stream.Start(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("could not run stream instance: %v", err)
	}

	var startStreamErr error
	defer func() {
		if startStreamErr != nil {
			if err := stream.Stop(ctx); err != nil {
				logrus.Errorf("could not stop container: %v", err)
			}
		}
	}()

	tcpAddress := fmt.Sprintf("%s:%d", streamLaunchInfo.IP, streamLaunchInfo.Port)
	rpcClient, err := connectToStream(tcpAddress, defaultConnectToStreamRetryCount)
	if err != nil {
		startStreamErr = fmt.Errorf("could not connect to the running stream: %v", err)
		return nil, nil, startStreamErr
	}

	return rpcClient, streamLaunchInfo, nil
}

func connectToStream(address string, tryCount int) (*rpc.Client, error) {
	for i := 0; i < tryCount; i++ {
		client, err := jsonrpc.Dial("tcp", address)
		if err == nil {
			return client, nil
		}

		logrus.Warnf("could not connect to the stream: %s %v", address, err)

		if i != tryCount {
			time.Sleep(defaultConnectToStreamRetryTimeout)
		}
	}

	return nil, errors.New("connection timeout")
}

func buildStreamOwnerInfo(info *streamInfo, accessToken string) *api.StreamOwnerInfo {
	ownerInfo := api.StreamOwnerInfo{
		UUID:        info.UUID,
		Name:        info.Name,
		Description: info.Description,
		JoinPolicy:  info.Join.Policy,
		JoinCode:    info.Join.Code,
		Port:        info.Port,
		IP:          info.IP,
		LaunchMode:  info.LaunchMode,
		Host: api.HostInfo{
			UUID:     info.Host.UUID,
			Username: info.Host.Username,
			AvatarID: info.Host.AvatarID,
			IP:       info.Host.IP,
		},
		StartedAt: info.StartedAt,
	}

	if accessToken != "" {
		ownerInfo.Auth = &api.StreamAuthInfo{
			AccessToken: accessToken,
			Type:        defaultStreamTokenType,
		}
	}

	return &ownerInfo
}
