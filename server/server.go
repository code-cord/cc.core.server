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
	"sync"
	"time"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler"
	"github.com/code-cord/cc.core.server/stream"
	"github.com/code-cord/cc.core.server/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultServerHost                  = "127.0.0.1"
	defaultStreamHost                  = "127.0.0.1"
	defaultServerFolder                = ".__data"
	defaultConnectToStreamRetryCount   = 3
	defaultConnectToStreamRetryTimeout = 2 * time.Second
)

// Server represents code-cord server implementation model.
type Server struct {
	opts       Options
	httpServer *http.Server
	log        *logrus.Logger
	streams    sync.Map
}

type streamInfo struct {
	api.Stream
	cfg      api.StreamConfig
	client   *rpc.Client
	hostUUID string
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
		log: logrus.New(),
	}
	if opts.LogLevel != "" {
		s.log.SetLevel(opts.logLevel)
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

	s.log.Infof("server started at %s", s.httpServer.Addr)

	if certFile, kFile := s.opts.TLSCertFile, s.opts.TLSKeyFile; certFile != "" && kFile != "" {
		err = s.httpServer.ListenAndServeTLS(certFile, kFile)
		return
	}

	err = s.httpServer.ListenAndServe()

	return
}

// Stop stops the running server.
func (s *Server) Stop(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("could not stop http server: %v", err)
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
	if cfg.Launch.PreferredIP == "" {
		cfg.Launch.PreferredIP = defaultStreamHost
	}
	if cfg.Launch.PreferredPort == 0 {
		port, err := util.FreePort(cfg.Launch.PreferredIP)
		if err != nil {
			return nil, fmt.Errorf("could not find free port to run stream: %v", err)
		}
		cfg.Launch.PreferredPort = port
	}

	tcpAddress := fmt.Sprintf("%s:%d", cfg.Launch.PreferredIP, cfg.Launch.PreferredPort)

	var (
		stream api.Stream
		err    error
	)
	streamUUID := uuid.New().String()
	switch cfg.Launch.Mode {
	case api.StreamLaunchModeSingletonApp:
		stream, err = s.newStandaloneStream(ctx, tcpAddress)
	case api.StreamLaunchModeDockerContainer:
		// TODO: implement
	default:
		return nil, fmt.Errorf("invalid launch mode: %v", cfg.Launch.Mode)
	}
	if err != nil {
		return nil, err
	}

	if err := stream.Start(ctx); err != nil {
		return nil, fmt.Errorf("could not run stream instance: %v", err)
	}

	var startStreamErr error
	defer func() {
		if startStreamErr != nil {
			stream.Stop(ctx)
		}
	}()

	client, err := connectToStream(tcpAddress, defaultConnectToStreamRetryCount)
	if err != nil {
		startStreamErr = fmt.Errorf("could not connect to the running stream: %v", err)
		return nil, startStreamErr
	}

	hostUUID := uuid.New().String()
	s.streams.Store(streamUUID, streamInfo{
		Stream:   stream,
		cfg:      cfg,
		client:   client,
		hostUUID: hostUUID,
	})

	return buildStreamOwnerInfo(&cfg, streamUUID, hostUUID), nil
}

func (s *Server) newStandaloneStream(ctx context.Context, tcpAddress string) (
	api.Stream, error) {
	standaloneStream := stream.NewStandaloneStream(stream.StandaloneStreamConfig{
		TCPAddress: tcpAddress,
		BinPath:    s.opts.BinFolder,
	})

	return &standaloneStream, nil
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

func buildStreamOwnerInfo(
	cfg *api.StreamConfig, streamUUID, hostUUID string) *api.StreamOwnerInfo {
	return &api.StreamOwnerInfo{
		UUID:        streamUUID,
		Name:        cfg.Name,
		Description: cfg.Description,
		JoinPolicy:  cfg.Join.JoinPolicy,
		JoinCode:    cfg.Join.JoinCode,
		Port:        cfg.Launch.PreferredPort,
		IP:          cfg.Launch.PreferredIP,
		LaunchMode:  cfg.Launch.Mode,
		Host: api.HostInfo{
			UUID:     hostUUID,
			Username: cfg.Host.Username,
			AvatarID: cfg.Host.AvatarID,
			IP:       cfg.Host.IP,
		},
	}
}

///////////////
func (s *Server) NewStream2() {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        "code-cord.stream",
		ExposedPorts: nat.PortSet{"30309": struct{}{}},
	}, &container.HostConfig{

		PortBindings: map[nat.Port][]nat.PortBinding{
			nat.Port("30309"): {
				{
					HostPort: "30309",
				},
			},
		},
	}, nil, nil, "ssasaasdasdsss")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	//time.Sleep(10 * time.Second)

	client, err := jsonrpc.Dial("tcp", "0.0.0.0:30309")
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

func (s *Server) PPP() {
	client, err := jsonrpc.Dial("tcp", "0.0.0.0:30303")
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
