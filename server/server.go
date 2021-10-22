package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc/jsonrpc"
	"os"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler"
	"github.com/code-cord/cc.core.server/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
)

const (
	defaultServerHost = "127.0.0.1"
)

// Server represents code-cord server implementation model.
type Server struct {
	opts       Options
	httpServer *http.Server
	log        *logrus.Logger
}

// New returns new Server instance.
func New(opt ...Option) Server {
	opts := newServerOptions(opt...)

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

///////////////
func (s *Server) NewStream() {
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

	return opts
}

func defaultServerAddress() string {
	freePort, err := util.FreePort(defaultServerHost)
	if err != nil {
		logrus.Fatalf("could not find any free port: %v", err)
	}

	return fmt.Sprintf("%s:%d", defaultServerHost, freePort)
}
