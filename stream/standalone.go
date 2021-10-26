package stream

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"runtime"
	"time"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/util"
)

const (
	defaultStreamBin       = "stream"
	defaultStandaloneAppIP = "127.0.0.1"
)

// StandaloneStream represents stream as standalone running app implementation model.
type StandaloneStream struct {
	preferedIP    string
	preferedPort  int
	binPath       string
	binCmd        *exec.Cmd
	interruptChan chan error
}

// StandaloneStreamConfig represents standalone stream configuration model.
type StandaloneStreamConfig struct {
	PreferedIP   string
	PreferedPort int
	BinPath      string
}

// NewStandaloneStream returns new standalone stream instance.
func NewStandaloneStream(cfg StandaloneStreamConfig) *StandaloneStream {
	return &StandaloneStream{
		preferedIP:    cfg.PreferedIP,
		preferedPort:  cfg.PreferedPort,
		binPath:       cfg.BinPath,
		interruptChan: make(chan error),
	}
}

// Start starts standalone stream.
func (s *StandaloneStream) Start(ctx context.Context) (*api.StartStreamInfo, error) {
	if s.preferedIP == "" {
		s.preferedIP = defaultStandaloneAppIP
	}

	if s.preferedPort == 0 {
		port, err := util.FreePort(s.preferedIP)
		if err != nil {
			return nil, fmt.Errorf("could not find free port to run stream: %v", err)
		}
		s.preferedPort = port
	}

	tcpAddress := fmt.Sprintf("%s:%d", s.preferedIP, s.preferedPort)
	streamPath := resolveBinPath(s.binPath, defaultStreamBin)
	s.binCmd = exec.Command(streamPath, "-addr", tcpAddress)

	if err := s.binCmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		err := s.binCmd.Wait()
		if err != nil {
			s.interruptChan <- err
		}
	}()

	time.Sleep(time.Second)

	return &api.StartStreamInfo{
		IP:   s.preferedIP,
		Port: s.preferedPort,
	}, nil
}

// Stop stops running stream.
func (s *StandaloneStream) Stop(ctx context.Context) error {
	return s.binCmd.Process.Kill()
}

// InterruptNotification returns an error when the stream has been interrupted.
func (s *StandaloneStream) InterruptNotification() <-chan error {
	return s.interruptChan
}

func resolveBinPath(binFolder, binName string) string {
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	return path.Join(binFolder, binName)
}
