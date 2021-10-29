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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/code-cord/cc.core.server/service"
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
	service.Stream
	rpcClient           *rpc.Client
	pendingParticipants *sync.Map
	rsaKeys             *rsaKeys
}

type streamInfo struct {
	UUID        string                   `json:"uuid"`
	Name        string                   `json:"name"`
	Description string                   `json:"desc,omitempty"`
	IP          string                   `json:"ip"`
	Port        int                      `json:"port"`
	LaunchMode  service.StreamLaunchMode `json:"mode"`
	StartedAt   time.Time                `json:"startedAt"`
	FinishedAt  *time.Time               `json:"finishedAt,omitempty"`
	Subject     string                   `json:"sub,omitempty"`
	Status      service.StreamStatus     `json:"status"`
	Join        streamJoinInfo           `json:"join"`
	Host        streamHostInfo           `json:"host"`
}

type streamJoinInfo struct {
	Code   string             `json:"code,omitempty"`
	Policy service.JoinPolicy `json:"policy"`
}

type streamHostInfo struct {
	UUID     string `json:"uuid"`
	Username string `json:"name"`
	AvatarID string `json:"avatar,omitempty"`
	IP       string `json:"ip"`
}

type sortFn func(i, j int) bool

// NewStream starts a new stream.
func (s *Server) NewStream(ctx context.Context, cfg service.StreamConfig) (
	*service.StreamOwnerInfo, error) {
	// generate stream access keys.
	keys, err := generateRSAKeys()
	if err != nil {
		return nil, fmt.Errorf("could not generate stream access keys: %v", err)
	}

	if cfg.Launch.Mode == "" {
		cfg.Launch.Mode = service.StreamLaunchModeStandaloneApp
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
		Status:      service.StreamStatusRunning,
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
func (s *Server) StreamInfo(
	ctx context.Context, streamUUID string) (*service.StreamPublicInfo, error) {
	streamRV := s.streamStorage.Default().Load(streamUUID)
	if streamRV == nil {
		return nil, os.ErrNotExist
	}

	var info streamInfo
	if err := streamRV.Decode(&info, json.Unmarshal); err != nil {
		return nil, fmt.Errorf("could not decode stream data: %v", err)
	}

	return &service.StreamPublicInfo{
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
func (s *Server) PatchStream(
	ctx context.Context, streamUUID string, cfg service.PatchStreamConfig) (
	*service.StreamOwnerInfo, error) {
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
	*service.AuthInfo, error) {
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

	return &service.AuthInfo{
		AccessToken: token,
		Type:        defaultStreamTokenType,
	}, nil
}

// StreamList returns stream list.
func (s *Server) StreamList(ctx context.Context, filter service.StreamFilter) (
	*service.StreamList, error) {
	cursor, err := s.streamStorage.Default().All()
	if err != nil {
		return nil, fmt.Errorf("could not fetch streams from storage: %v", err)
	}

	streams := make([]service.StreamInfo, 0, s.streamStorage.Default().Size())
	for rv, hasNext := cursor.First(); hasNext; rv, hasNext = cursor.Next() {
		var stream streamInfo
		if err := rv.Decode(&stream, json.Unmarshal); err != nil {
			return nil, fmt.Errorf("could not parse stream info: %v", err)
		}

		if isStreamFitsFilter(&stream, &filter) {
			streams = append(streams, service.StreamInfo{
				UUID:        stream.UUID,
				Name:        stream.Name,
				Description: stream.Description,
				IP:          stream.IP,
				Port:        stream.Port,
				LaunchMode:  stream.LaunchMode,
				StartedAt:   stream.StartedAt,
				FinishedAt:  stream.FinishedAt,
				Status:      stream.Status,
				Join: service.StreamJoinPolicyConfig{
					JoinPolicy: stream.Join.Policy,
					JoinCode:   stream.Join.Code,
				},
				Host: service.HostInfo{
					UUID:     stream.Host.UUID,
					Username: stream.Host.Username,
					AvatarID: stream.Host.AvatarID,
					IP:       stream.Host.IP,
				},
			})
		}
	}

	filterStreams(streams, filter.SortBy, filter.SortOrder)

	var filteredStreams []service.StreamInfo
	startIndex := (filter.Page - 1) * filter.PageSize
	endIndex := startIndex + filter.PageSize

	if startIndex < len(streams) {
		filteredStreams = streams[startIndex:]
	}

	if endIndex < len(streams) {
		filteredStreams = filteredStreams[:endIndex-startIndex]
	}

	total := len(streams)
	return &service.StreamList{
		Streams:  filteredStreams,
		Count:    len(filteredStreams),
		Total:    total,
		PageSize: filter.PageSize,
		Page:     filter.Page,
		HasNext:  total > filter.Page*filter.PageSize,
	}, nil
}

func (s *Server) newStreamHandler(cfg service.StreamConfig, streamUUID string) (service.Stream, error) {
	switch cfg.Launch.Mode {
	case service.StreamLaunchModeStandaloneApp:
		return stream.NewStandaloneStream(stream.StandaloneStreamConfig{
			PreferedIP:   cfg.Launch.PreferredIP,
			PreferedPort: cfg.Launch.PreferredPort,
			BinPath:      s.opts.BinFolder,
		}), nil
	case service.StreamLaunchModeDockerContainer:
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
	stream.Status = service.StreamStatusFinished
	if err := s.streamStorage.Default().Store(streamUUID, stream, json.Marshal); err != nil {
		logrus.Errorf("could not store %s stream data to finish: %v", streamUUID, err)
	}
}

func isStreamFitsFilter(stream *streamInfo, filter *service.StreamFilter) bool {
	// filter by search term.
	if !strings.Contains(stream.Name, filter.SearchPhrase) &&
		!strings.Contains(stream.Description, filter.SearchPhrase) {
		return false
	}

	// filter by launch mode.
	found := len(filter.LaunchModes) == 0
	for i := range filter.LaunchModes {
		if filter.LaunchModes[i] == stream.LaunchMode {
			found = true
			break
		}
	}
	if !found {
		return false
	}

	// filter by status.
	found = len(filter.Statuses) == 0
	for i := range filter.Statuses {
		if filter.Statuses[i] == stream.Status {
			found = true
			break
		}
	}

	return found
}

func filterStreams(
	streams []service.StreamInfo,
	sortBy service.StreamSortByField,
	sortOrder service.StreamSortOrder) {
	var sortFunc sortFn
	switch sortBy {
	case service.StreamSortByFieldUUID:
		sortFunc = filterStreamsByUUID(streams, sortOrder)
	case service.StreamSortByFieldName:
		sortFunc = filterStreamsByName(streams, sortOrder)
	case service.StreamSortByFieldLaunchMode:
		sortFunc = filterStreamsByLaunchMode(streams, sortOrder)
	case service.StreamSortByFieldStarted:
		sortFunc = filterStreamsByStartedDate(streams, sortOrder)
	case service.StreamSortByFieldStatus:
		sortFunc = filterStreamsByStatus(streams, sortOrder)
	}

	if sortFunc != nil {
		sort.Slice(streams, sortFunc)
	}
}

func filterStreamsByUUID(streams []service.StreamInfo, sortOrder service.StreamSortOrder) sortFn {
	return func(i, j int) bool {
		if sortOrder == service.StreamSortOrderAsc {
			return streams[i].UUID < streams[j].UUID
		}

		return streams[i].UUID > streams[j].UUID
	}
}

func filterStreamsByName(streams []service.StreamInfo, sortOrder service.StreamSortOrder) sortFn {
	return func(i, j int) bool {
		if sortOrder == service.StreamSortOrderAsc {
			return streams[i].Name < streams[j].Name
		}

		return streams[i].Name > streams[j].Name
	}
}

func filterStreamsByLaunchMode(
	streams []service.StreamInfo, sortOrder service.StreamSortOrder) sortFn {
	return func(i, j int) bool {
		if sortOrder == service.StreamSortOrderAsc {
			return string(streams[i].LaunchMode) < string(streams[j].LaunchMode)
		}

		return string(streams[i].LaunchMode) > string(streams[j].LaunchMode)
	}
}

func filterStreamsByStartedDate(
	streams []service.StreamInfo, sortOrder service.StreamSortOrder) sortFn {
	return func(i, j int) bool {
		if sortOrder == service.StreamSortOrderAsc {
			return streams[i].StartedAt.Before(streams[j].StartedAt)
		}

		return streams[i].StartedAt.After(streams[j].StartedAt)
	}
}

func filterStreamsByStatus(
	streams []service.StreamInfo, sortOrder service.StreamSortOrder) sortFn {
	return func(i, j int) bool {
		if sortOrder == service.StreamSortOrderAsc {
			return string(streams[i].Status) < string(streams[j].Status)
		}

		return string(streams[i].Status) > string(streams[j].Status)
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

func startStreamAndConnect(ctx context.Context, stream service.Stream) (
	*rpc.Client, *service.StartStreamInfo, error) {
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

func buildStreamOwnerInfo(info *streamInfo, accessToken string) *service.StreamOwnerInfo {
	ownerInfo := service.StreamOwnerInfo{
		UUID:        info.UUID,
		Name:        info.Name,
		Description: info.Description,
		JoinPolicy:  info.Join.Policy,
		JoinCode:    info.Join.Code,
		Port:        info.Port,
		IP:          info.IP,
		LaunchMode:  info.LaunchMode,
		Host: service.HostInfo{
			UUID:     info.Host.UUID,
			Username: info.Host.Username,
			AvatarID: info.Host.AvatarID,
			IP:       info.Host.IP,
		},
		StartedAt: info.StartedAt,
	}

	if accessToken != "" {
		ownerInfo.Auth = &service.AuthInfo{
			AccessToken: accessToken,
			Type:        defaultStreamTokenType,
		}
	}

	return &ownerInfo
}
