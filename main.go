package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/code-cord/cc.core.server/server"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

/*
import (
	"bufio"
	"log"
	"net/rpc/jsonrpc"
	"os"
)

type Reply struct {
	Data string
}

func main() {
	client, err := jsonrpc.Dial("tcp", "localhost:12345")
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
*/

const (
	defaultStreamPrefixContainer = "code-cord.stream"
	codeCordBinPathEnv           = "CODE_CORD_PATH"
	defaultStreamImage           = "code-cord.stream"
)

//go:embed server.json
var serverInfo []byte

type serverConfig struct {
	address               string
	tlsCertFilePath       string
	tlsKeyFilePath        string
	logLevel              string
	streamContainerPrefix string
	streamImage           string
	dataFolder            string
	maxAvatarSize         int64
	binariesPath          string
}

func main() {
	var cfg serverConfig

	app := &cli.App{
		Name:  "code-cord-server",
		Usage: "manage code-cord stream server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name: "address",
				Aliases: []string{
					"addr",
					"a",
				},
				Usage:       "Server listen and serve address",
				DefaultText: "127.0.0.1:0",
				Destination: &cfg.address,
				Required:    false,
			},
			&cli.PathFlag{
				Name: "tls-cert",
				Aliases: []string{
					"cert",
				},
				Usage:       "TLS cert file path (for https connections)",
				Required:    false,
				TakesFile:   true,
				Destination: &cfg.tlsCertFilePath,
			},
			&cli.PathFlag{
				Name: "tls-key",
				Aliases: []string{
					"key",
				},
				Usage:       "TLS key file path (for https connections)",
				Required:    false,
				TakesFile:   true,
				Destination: &cfg.tlsKeyFilePath,
			},
			&cli.StringFlag{
				Name: "log",
				Aliases: []string{
					"lvl",
					"level",
					"logger",
				},
				Usage:       "Log level (\"panic\", \"fatal\", \"error\", \"warn\", \"info\", \"debug\" or \"trace\")",
				Required:    false,
				DefaultText: "info",
				Destination: &cfg.logLevel,
			},
			&cli.StringFlag{
				Name: "stream-container-prefix",
				Aliases: []string{
					"prefix",
					"container-prefix",
				},
				Usage:       "Stream container prefix for streams running inside docker containers",
				Required:    false,
				DefaultText: defaultStreamPrefixContainer,
				Destination: &cfg.streamContainerPrefix,
				Value:       defaultStreamPrefixContainer,
			},
			&cli.PathFlag{
				Name: "data",
				Aliases: []string{
					"folder",
					"data-folder",
				},
				Usage:       "Data folder to store server data",
				Required:    false,
				Destination: &cfg.tlsCertFilePath,
				DefaultText: "current/dir/__data/...",
			},
			&cli.Int64Flag{
				Name:        "avatar-size",
				Usage:       "Maximum acceptable size of the incoming avatar image (in bytes)",
				Required:    false,
				Destination: &cfg.maxAvatarSize,
				DefaultText: "no restrictions",
			},
			&cli.PathFlag{
				Name: "cc-path",
				Aliases: []string{
					"bin",
					"code-cord-bin",
				},
				Usage:       "Folder path to code-cord binaries",
				Required:    false,
				Destination: &cfg.binariesPath,
				EnvVars: []string{
					codeCordBinPathEnv,
				},
			},
			&cli.StringFlag{
				Name: "stream-image",
				Aliases: []string{
					"img",
					"stream-img",
				},
				Usage:       "Stream image to run inside the container",
				Required:    false,
				DefaultText: defaultStreamImage,
				Destination: &cfg.streamImage,
				Value:       defaultStreamImage,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatalf("could not start app: %v", err)
	}

	s, err := newServer(cfg)
	if err != nil {
		logrus.Fatalf("could not create server instance: %v", err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logrus.Warn("server is shutting down...")
		if err := s.Stop(context.Background()); err != nil {
			logrus.Errorf("could not stop server: %v", err)
		}
	}()

	s.Run(context.Background())
}

func newServer(cfg serverConfig) (*server.Server, error) {
	var info map[string]interface{}
	if err := json.Unmarshal(serverInfo, &info); err != nil {
		return nil, err
	}

	s := server.New(
		// Default configuration.
		server.Name(info["name"].(string)),
		server.Description(info["description"].(string)),
		server.Version(info["version"].(string)),
		server.Meta(info["meta"].(map[string]interface{})),
		// CLI configuration.
		server.Address(cfg.address),
		server.TLS(cfg.tlsCertFilePath, cfg.tlsKeyFilePath),
		server.LogLevel(cfg.logLevel),
		server.StreamContainerPrefix(cfg.streamContainerPrefix),
		server.StreamImage(cfg.streamImage),
		server.DataFolder(cfg.dataFolder),
		server.MaxAvatarSize(cfg.maxAvatarSize),
		server.BinFolder(cfg.binariesPath),
	)

	return &s, nil
}
