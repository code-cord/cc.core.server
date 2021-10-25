package stream

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"runtime"
	"time"
)

const (
	defaultStreamBin = "stream"
)

// StandaloneStream represents stream as standalone running app implementation model.
type StandaloneStream struct {
	tcpAddress string
	binPath    string
	binCmd     *exec.Cmd
}

// StandaloneStreamConfig represents standalone stream configuration model.
type StandaloneStreamConfig struct {
	TCPAddress string
	BinPath    string
}

// NewStandaloneStream returns new standalone stream instance.
func NewStandaloneStream(cfg StandaloneStreamConfig) StandaloneStream {
	return StandaloneStream{
		tcpAddress: cfg.TCPAddress,
		binPath:    cfg.BinPath,
	}
}

// Start starts standalone stream.
func (s *StandaloneStream) Start(ctx context.Context) error {
	streamPath := resolveBinPath(s.binPath, defaultStreamBin)
	s.binCmd = exec.Command(streamPath, "-addr", s.tcpAddress)

	if err := s.binCmd.Start(); err != nil {
		return err
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

	return <-errChan
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
