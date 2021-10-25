package stream

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
)

const (
	defaultDockerContainerHostIP = "0.0.0.0"
)

// DockerContainerStream represents stream as a running docker container implementation model.
type DockerContainerStream struct {
	streamUUID      string
	containerPrefix string
	dockerImage     string
	containerID     string
	preferedPort    int
	preferedIP      string
}

// DockerContainerStreamConfig represents docker container stream configuration model.
type DockerContainerStreamConfig struct {
	StreamUUID      string
	ContainerPrefix string
	DockerImage     string
	PreferedPort    int
	PreferedIP      string
}

// NewDockerContainerStream returns new stream as docker container instance.
func NewDockerContainerStream(cfg DockerContainerStreamConfig) *DockerContainerStream {
	return &DockerContainerStream{
		streamUUID:      cfg.StreamUUID,
		containerPrefix: cfg.ContainerPrefix,
		dockerImage:     cfg.DockerImage,
		preferedPort:    cfg.PreferedPort,
		preferedIP:      cfg.PreferedIP,
	}
}

// Start starts docker container stream.
func (s *DockerContainerStream) Start(ctx context.Context) (*api.StartStreamInfo, error) {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		return nil, fmt.Errorf("could not init docker cli client: %v", err)
	}

	if s.preferedIP == "" {
		s.preferedIP = defaultDockerContainerHostIP
	}

	if s.preferedPort == 0 {
		port, err := util.FreePort(s.preferedIP)
		if err != nil {
			return nil, fmt.Errorf("could not find free port to run container: %v", err)
		}
		s.preferedPort = port
	}

	tcpAddress := fmt.Sprintf("%s:%d", s.preferedIP, s.preferedPort)
	portStr := strconv.Itoa(s.preferedPort)
	containerCfg := container.Config{
		Image:        s.dockerImage,
		ExposedPorts: nat.PortSet{nat.Port(portStr): struct{}{}},
		Cmd: strslice.StrSlice{
			"/start", "-addr", tcpAddress,
		},
	}
	containerHostCfg := container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			nat.Port(portStr): {
				{
					HostPort: portStr,
					HostIP:   s.preferedIP,
				},
			},
		},
	}
	containerName := fmt.Sprintf("%s-%s", s.containerPrefix, s.streamUUID)

	containerBody, err := cli.ContainerCreate(
		ctx, &containerCfg, &containerHostCfg, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("could not create docker container: %v", err)
	}

	s.containerID = containerBody.ID
	for i := range containerBody.Warnings {
		logrus.Warn(containerBody.Warnings[i])
	}

	if err := cli.ContainerStart(ctx, s.containerID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("could not start docker container: %v", err)
	}

	time.Sleep(time.Second)

	return &api.StartStreamInfo{
		IP:   s.preferedIP,
		Port: s.preferedPort,
	}, nil
}

// Stop stops running stream.
func (s *DockerContainerStream) Stop(ctx context.Context) error {
	cli, err := client.NewClientWithOpts()
	if err != nil {
		return fmt.Errorf("could not init docker cli client: %v", err)
	}

	return cli.ContainerStop(ctx, s.containerID, nil)
}
