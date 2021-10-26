package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler"
	"github.com/code-cord/cc.core.server/stream"
	"github.com/code-cord/cc.core.server/util"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultServerHost                  = "127.0.0.1"
	defaultServerFolder                = ".__data"
	defaultConnectToStreamRetryCount   = 3
	defaultConnectToStreamRetryTimeout = 2 * time.Second
)

// Server represents code-cord server implementation model.
type Server struct {
	opts       Options
	httpServer *http.Server
	streams    *sync.Map
}

type streamInfo struct {
	api.Stream
	name        string
	description string
	ip          string
	port        int
	join        api.StreamJoinPolicyConfig
	rcpClient   *rpc.Client
	startedAt   time.Time
	hostInfo    api.HostInfo
	launchMode  api.StreamLaunchMode
}

// New returns new Server instance.
func New(opt ...Option) Server {
	opts := newServerOptions(opt...)

	avatarService, err := NewAvatarService(AvatarServiceConfig{
		DataFolder:  opts.DataFolder,
		MaxFileSize: opts.MaxAvatarSize,
	})
	if err != nil {
		logrus.Fatalf("could not init avatar service: %v", err)
	}

	s := Server{
		opts: opts,
		httpServer: &http.Server{
			Addr: opts.Address,
		},
		streams: new(sync.Map),
	}
	if opts.LogLevel != "" {
		logrus.SetLevel(opts.logLevel)
	}
	s.httpServer.Handler = handler.New(handler.Config{
		Server: &s,
		Avatar: avatarService,
	})

	return s
}

// Run runs server.
func (s *Server) Run(ctx context.Context) (err error) {
	defer func() {
		if err == http.ErrServerClosed {
			err = nil
		}

		err = fmt.Errorf("could not serve http server: %v", err)
	}()

	logrus.Infof("server started at %s", s.httpServer.Addr)

	if certFile, kFile := s.opts.TLSCertFile, s.opts.TLSKeyFile; certFile != "" && kFile != "" {
		err = s.httpServer.ListenAndServeTLS(certFile, kFile)
		return
	}

	err = s.httpServer.ListenAndServe()

	return
}

// Stop stops the running server.
func (s *Server) Stop(ctx context.Context) error {
	errs := make([]string, 0)
	s.streams.Range(func(key, value interface{}) bool {
		s := value.(streamInfo)
		if err := s.rcpClient.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("could not close %s stream connection: %v", key, err))
		}

		if err := s.Stream.Stop(ctx); err != nil {
			errs = append(errs, fmt.Sprintf("could not stop %s stream: %v", key, err))
		}

		return true
	})

	if err := s.httpServer.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Sprintf("could not stop http server: %v", err))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, ";"))
	}

	return nil
}

// Info returns server public info.
func (s *Server) Info() api.ServerInfo {
	return api.ServerInfo{
		Name:        s.opts.Name,
		Description: s.opts.Description,
		Version:     s.opts.Version,
		Meta:        s.opts.Meta,
	}
}

// Ping pings server.
func (s *Server) Ping(ctx context.Context) error {
	return nil
}

// NewStream starts a new stream.
func (s *Server) NewStream(ctx context.Context, cfg api.StreamConfig) (
	*api.StreamOwnerInfo, error) {
	if cfg.Launch.Mode == "" {
		cfg.Launch.Mode = api.StreamLaunchModeSingletonApp
	}

	var streamHandler api.Stream
	streamUUID := uuid.New().String()
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
			UUID:     uuid.New().String(),
			Username: cfg.Host.Username,
			AvatarID: cfg.Host.AvatarID,
			IP:       cfg.Host.IP,
		},
		launchMode: cfg.Launch.Mode,
	}
	s.streams.Store(streamUUID, newStreamInfo)

	return buildStreamOwnerInfo(&newStreamInfo, streamUUID), nil
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

func (s *Server) listenStreamInterruptEvent(streamUUID string, intChan <-chan error) {
	err := <-intChan
	if err != nil {
		logrus.Errorf("stream %s has been interrupted: %v", streamUUID, err)
	}

	s.streams.Delete(streamUUID)
}

func connectToStream(address string, tryCount int) (*rpc.Client, error) {
	for i := 0; i < tryCount; i++ {
		client, err := jsonrpc.Dial("tcp", address)
		if err == nil {
			return client, nil
		}

		logrus.Warnf("could not connect to the stream: %s %v", address, err)

		time.Sleep(defaultConnectToStreamRetryTimeout)
	}

	return nil, errors.New("connection timeout")
}

func buildStreamOwnerInfo(info *streamInfo, streamUUID string) *api.StreamOwnerInfo {
	return &api.StreamOwnerInfo{
		UUID:        streamUUID,
		Name:        info.name,
		Description: info.description,
		JoinPolicy:  info.join.JoinPolicy,
		JoinCode:    info.join.JoinCode,
		Port:        info.port,
		IP:          info.ip,
		LaunchMode:  info.launchMode,
		Host:        info.hostInfo,
		StartedAt:   info.startedAt,
	}
}

///////////////

func (s *Server) PPP() {
	client, err := jsonrpc.Dial("tcp", "0.0.0.0:30300")
	if err != nil {
		log.Fatal(err)
	}
	in := bufio.NewReader(os.Stdin)
	for {
		line, _, err := in.ReadLine()
		if err != nil {
			log.Fatal(err)
		}
		var reply Reply
		err = client.Call("Listener.GetLine", line, &reply)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Reply: %v, Data: %v", reply, reply.Data)
	}
}

type Reply struct {
	Data string
}

////////////////

func newServerOptions(opt ...Option) Options {
	var opts Options

	for _, o := range opt {
		o(&opts)
	}

	if opts.Address == "" {
		opts.Address = defaultServerAddress()
	}

	if opts.LogLevel != "" {
		lvl, err := logrus.ParseLevel(opts.LogLevel)
		if err != nil {
			logrus.Errorf("could not set log level: %v", err)
			logrus.Infof("default log level will be used: \"%s\"", logrus.InfoLevel)
			lvl = logrus.InfoLevel
		}
		opts.logLevel = lvl
	}

	if opts.DataFolder == "" {
		dir, err := os.Getwd()
		if err != nil {
			logrus.Fatalf("could not detect working directory path: %v", err)
		}
		opts.DataFolder = path.Join(dir, defaultServerFolder)
		if err := os.MkdirAll(opts.DataFolder, 0700); err != nil && !os.IsExist(err) {
			logrus.Panicf("could not create server folders: %v", err)
		}
	}

	return opts
}

func defaultServerAddress() string {
	freePort, err := util.FreePort(defaultServerHost)
	if err != nil {
		logrus.Fatalf("could not find any free port: %v", err)
	}

	return fmt.Sprintf("%s:%d", defaultServerHost, freePort)
}
