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

const (
	tlsCertFilePathEnv              = "CODE_CORD_TLS_CERT"
	tlsKeyFilePathEnv               = "CODE_CORD_TLS_KEY"
	dataFolderPathEnv               = "CODE_CORD_DATA"
	codeCordBinPathEnv              = "CODE_CORD_PATH"
	codeCordServerPublicKeyPathEnv  = "CODE_CORD_SERVER_PUBLIC_KEY"
	codeCordServerPrivateKeyPathEnv = "CODE_CORD_SERVER_PRIVATE_KEY"

	defaultStreamPrefixContainer = "code-cord.stream"
	defaultStreamImage           = "code-cord.stream"
)

//go:embed build.json
var buildInfo []byte

type serverConfig struct {
	address                 string
	tlsCertFilePath         string
	tlsKeyFilePath          string
	logLevel                string
	streamContainerPrefix   string
	streamImage             string
	streamImageRegistryAuth string
	pullImageOnStartup      bool
	dataFolder              string
	maxAvatarSize           int64
	withSecurityCheck       bool
	securityPublicKeyPath   string
	securityPrivateKeyPath  string
	binariesPath            string
}

func main() {
	var cfg serverConfig

	app := &cli.App{
		Name:  "code-cord-server",
		Usage: "manage code-cord stream server",
		Action: func(c *cli.Context) error {
			return nil
		},
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
				Name:      "tls-cert",
				Usage:     "TLS cert file path (for https connections)",
				Required:  false,
				TakesFile: true,
				EnvVars: []string{
					tlsCertFilePathEnv,
				},
				Destination: &cfg.tlsCertFilePath,
			},
			&cli.PathFlag{
				Name:      "tls-key",
				Usage:     "TLS key file path (for https connections)",
				Required:  false,
				TakesFile: true,
				EnvVars: []string{
					tlsKeyFilePathEnv,
				},
				Destination: &cfg.tlsKeyFilePath,
			},
			&cli.StringFlag{
				Name: "log",
				Aliases: []string{
					"lvl",
					"level",
				},
				Usage:       "Log level (\"panic\", \"fatal\", \"error\", \"warn\", \"info\", \"debug\" or \"trace\")",
				Required:    false,
				DefaultText: logrus.InfoLevel.String(),
				Destination: &cfg.logLevel,
			},
			&cli.StringFlag{
				Name: "stream-container-prefix",
				Aliases: []string{
					"container-prefix",
				},
				Usage:       "Stream container prefix for streams running inside docker containers",
				Required:    false,
				DefaultText: defaultStreamPrefixContainer,
				Destination: &cfg.streamContainerPrefix,
				Value:       defaultStreamPrefixContainer,
			},
			&cli.StringFlag{
				Name: "stream-image",
				Aliases: []string{
					"img",
					"docker-img",
				},
				Usage:       "Stream docker image to run inside the container",
				Required:    false,
				DefaultText: defaultStreamImage,
				Destination: &cfg.streamImage,
				Value:       defaultStreamImage,
			},
			&cli.StringFlag{
				Name: "stream-image-registry-auth",
				Aliases: []string{
					"img-registry-auth",
				},
				Usage:       "The base64 encoded credentials for the stream container registry",
				Required:    false,
				Destination: &cfg.streamImageRegistryAuth,
			},
			&cli.BoolFlag{
				Name: "pull-image-on-startup",
				Aliases: []string{
					"pull-image",
				},
				Usage:       "Automatically pull stream docker image on server startup",
				Required:    false,
				Value:       false,
				DefaultText: "disabled",
				Destination: &cfg.pullImageOnStartup,
			},
			&cli.PathFlag{
				Name: "data-folder",
				Aliases: []string{
					"data",
				},
				Usage:       "Data folder to store server data with read and write permissions",
				Required:    false,
				Destination: &cfg.dataFolder,
				DefaultText: ".data folder in the current dir",
				EnvVars: []string{
					dataFolderPathEnv,
				},
			},
			&cli.Int64Flag{
				Name:        "avatar-size",
				Usage:       "Maximum acceptable size of the incoming avatar image (in bytes)",
				Required:    false,
				Destination: &cfg.maxAvatarSize,
				DefaultText: "no restrictions",
				Value:       -1,
			},
			&cli.PathFlag{
				Name: "bin-path",
				Aliases: []string{
					"bin",
				},
				Usage:       "Folder path to code-cord binaries (required only if stream will be running as standalone application)",
				Required:    false,
				Destination: &cfg.binariesPath,
				EnvVars: []string{
					codeCordBinPathEnv,
				},
				DefaultText: "current directory",
			},
			&cli.BoolFlag{
				Name: "with-security-check",
				Aliases: []string{
					"securely",
				},
				Usage:       "Enable server security check for incoming requests (eg. create/update/stop stream)",
				Required:    false,
				Value:       false,
				DefaultText: "disabled",
				Destination: &cfg.withSecurityCheck,
			},
			&cli.PathFlag{
				Name: "server-public-key",
				Aliases: []string{
					"pub-key",
				},
				Usage:       "Path to server public RSA key file",
				Required:    false,
				TakesFile:   true,
				Destination: &cfg.securityPublicKeyPath,
				EnvVars: []string{
					codeCordServerPublicKeyPathEnv,
				},
			},
			&cli.PathFlag{
				Name: "server-private-key",
				Aliases: []string{
					"priv-key",
				},
				Usage:       "Path to server private RSA key file",
				Required:    false,
				TakesFile:   true,
				Destination: &cfg.securityPrivateKeyPath,
				EnvVars: []string{
					codeCordServerPrivateKeyPathEnv,
				},
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

	if err := s.Run(context.Background()); err != nil {
		logrus.Fatalf("could not run server: %v", err)
	}
}

func newServer(cfg serverConfig) (*server.Server, error) {
	var info map[string]interface{}
	if err := json.Unmarshal(buildInfo, &info); err != nil {
		return nil, err
	}

	return server.New(
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
		server.StreamImageRegistryAuth(cfg.streamImageRegistryAuth),
		server.PullImageOnStartup(cfg.pullImageOnStartup),
		server.DataFolder(cfg.dataFolder),
		server.MaxAvatarSize(cfg.maxAvatarSize),
		server.BinFolder(cfg.binariesPath),
		server.ServerSecurityEnabled(cfg.withSecurityCheck),
		server.ServerPrivateKey(cfg.securityPrivateKeyPath),
		server.ServerPublicKey(cfg.securityPublicKeyPath),
	)
}
