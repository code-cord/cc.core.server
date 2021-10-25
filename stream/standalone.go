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
	preferedIP   string
	preferedPort int
	binPath      string
	binCmd       *exec.Cmd
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
		preferedIP:   cfg.PreferedIP,
		preferedPort: cfg.PreferedPort,
		binPath:      cfg.BinPath,
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

	errChan := make(chan error)
	go func() {
		err := s.binCmd.Wait()
		if err != nil {
			errChan <- fmt.Errorf("could not stream binary: %v", err)
		}
	}()

	go func() {
		time.Sleep(2 * time.Second)
		close(errChan)
	}()

	return &api.StartStreamInfo{
		IP:   s.preferedIP,
		Port: s.preferedPort,
	}, <-errChan
}

// Stop stops running stream.
func (s *StandaloneStream) Stop(ctx context.Context) error {
	return s.binCmd.Process.Kill()
}

func resolveBinPath(binFolder, binName string) string {
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	return path.Join(binFolder, binName)
}
