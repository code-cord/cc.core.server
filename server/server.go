package server

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/code-cord/cc.core.server/api"
	"github.com/code-cord/cc.core.server/handler"
	apiHandler "github.com/code-cord/cc.core.server/handler/api"
	"github.com/code-cord/cc.core.server/storage"
	"github.com/code-cord/cc.core.server/util"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
)

const (
	defaultServerHost             = "127.0.0.1"
	defaultAPIServerHost          = "127.0.0.1"
	defaultAPIServerPort          = 7070
	defaultServerFolder           = ".data"
	defaultStreamStorageName      = "stream.db"
	defaultAvatarStorageName      = "avatar.db"
	defaultParticipantStorageName = "participant.db"
	streamBucket                  = "stream"
	avatarBucket                  = "avatar"
	participantBucket             = "participant"
)

// Server represents code-cord server implementation model.
type Server struct {
	opts               Options
	httpServer         *http.Server
	apiHttpServer      *http.Server
	streams            *sync.Map
	streamStorage      *storage.Storage
	avatarStorage      *storage.Storage
	participantStorage *storage.Storage
}

type rsaKeys struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// New returns new Server instance.
func New(opt ...Option) (*Server, error) {
	opts, err := newServerOptions(opt...)
	if err != nil {
		return nil, fmt.Errorf("could not init server: %v", err)
	}

	streamDB, err := storage.New(storage.Config{
		DBPath:        path.Join(opts.DataFolder, defaultStreamStorageName),
		Buckets:       []string{streamBucket},
		DefaultBucket: streamBucket,
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to stream storage: %v", err)
	}

	avatarDB, err := storage.New(storage.Config{
		DBPath:        path.Join(opts.DataFolder, defaultAvatarStorageName),
		Buckets:       []string{avatarBucket},
		DefaultBucket: avatarBucket,
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to avatar storage: %v", err)
	}

	participantDB, err := storage.New(storage.Config{
		DBPath:        path.Join(opts.DataFolder, defaultParticipantStorageName),
		Buckets:       []string{participantBucket},
		DefaultBucket: participantBucket,
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to participant storage: %v", err)
	}

	s := Server{
		opts: *opts,
		httpServer: &http.Server{
			Addr: opts.Address,
		},
		apiHttpServer: &http.Server{
			Addr: fmt.Sprintf("%s:%d", defaultAPIServerHost, defaultAPIServerPort),
		},
		streams:            new(sync.Map),
		streamStorage:      streamDB,
		avatarStorage:      avatarDB,
		participantStorage: participantDB,
	}
	if opts.LogLevel != "" {
		logrus.SetLevel(opts.logLevel)
	}
	s.httpServer.Handler = handler.New(handler.Config{
		Server:               &s,
		SeverSecurityEnabled: s.opts.ServerSecurityEnabled,
		ServerPublicKey:      s.opts.publicKey,
	})
	s.apiHttpServer.Handler = apiHandler.New(apiHandler.Config{
		Server: &s,
	})

	return &s, nil
}

// Run runs server.
func (s *Server) Run(ctx context.Context) (err error) {
	// run API http server.
	go func() {
		logrus.Infof("starting API server at %s", s.apiHttpServer.Addr)
		if err := s.apiHttpServer.ListenAndServe(); err != http.ErrServerClosed {
			logrus.Fatalf("API server exited with error: %v", err)
		}
	}()

	// run core http server.
	logrus.Infof("starting server at %s", s.httpServer.Addr)

	defer func() {
		if err != http.ErrServerClosed {
			logrus.Fatalf("server exited with error: %v", err)
		}

		err = nil
	}()

	if s.opts.tlsEnabled {
		err = s.httpServer.ListenAndServeTLS(s.opts.TLSCertFile, s.opts.TLSKeyFile)
		return
	}

	err = s.httpServer.ListenAndServe()

	return
}

// Stop stops the running server.
func (s *Server) Stop(ctx context.Context) error {
	errs := make([]string, 0)
	s.streams.Range(func(key, value interface{}) bool {
		s.killStream(ctx, key.(string))

		return true
	})

	if err := s.httpServer.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Sprintf("could not stop http server: %v", err))
	}

	if err := s.apiHttpServer.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Sprintf("could not stop API http server: %v", err))
	}

	if err := s.avatarStorage.Close(); err != nil {
		errs = append(errs, fmt.Sprintf(
			"could not close connection to avatar storage: %v", err))
	}

	if err := s.streamStorage.Close(); err != nil {
		errs = append(errs, fmt.Sprintf(
			"could not close connection to stream storage: %v", err))
	}

	if err := s.participantStorage.Close(); err != nil {
		errs = append(errs, fmt.Sprintf(
			"could not close connection to participant storage: %v", err))
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

// NewServerToken generates new server auth token.
func (s *Server) NewServerToken(ctx context.Context, claims *jwt.StandardClaims) (
	*api.AuthInfo, error) {
	if s.opts.privateKey == nil {
		return nil, errors.New("server doesn't have RSA private key")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, *claims)
	tokenStr, err := token.SignedString(s.opts.privateKey)
	if err != nil {
		return nil, err
	}

	return &api.AuthInfo{
		AccessToken: tokenStr,
	}, nil
}

// StorageBackup creates backup of the provided storage.
func (s *Server) StorageBackup(
	ctx context.Context, storageName api.ServerStorage, w io.Writer) error {
	var storage *storage.Storage
	switch storageName {
	case api.ServerStorageAvatar:
		storage = s.avatarStorage
	case api.ServerStorageParticipant:
		storage = s.participantStorage
	case api.ServerStorageStream:
		storage = s.streamStorage
	default:
		return fmt.Errorf("invalid storage name: %s", storageName)
	}

	return storage.Backup(w)
}

func newServerOptions(opt ...Option) (*Options, error) {
	var opts Options

	for _, o := range opt {
		o(&opts)
	}

	if opts.Address == "" {
		opts.Address = defaultServerAddress()
	}

	if opts.TLSCertFile == "" && opts.TLSKeyFile != "" {
		return nil, fmt.Errorf("you must provide the TLS cert file along with the TLS key file")
	}
	if opts.TLSCertFile != "" && opts.TLSKeyFile == "" {
		return nil, fmt.Errorf("you must provide the TLS key file along with the TLS cert file")
	}
	opts.tlsEnabled = opts.TLSCertFile != "" && opts.TLSKeyFile != ""

	if opts.LogLevel != "" {
		lvl, err := logrus.ParseLevel(opts.LogLevel)
		if err != nil {
			logrus.Errorf("could not set log level: %v", err)
			logrus.Infof("default log level will be used: \"%s\"", logrus.InfoLevel)
			lvl = logrus.InfoLevel
		}
		opts.logLevel = lvl
	}

	if opts.PullImageOnStartup {
		cli, err := client.NewClientWithOpts()
		if err != nil {
			return nil, fmt.Errorf("could not init docker cli client: %v", err)
		}

		logrus.Infof("pulling %s docker image...", opts.StreamImage)

		_, err = cli.ImagePull(context.Background(), opts.StreamImage, types.ImagePullOptions{
			All:          true,
			RegistryAuth: opts.StreamImageRegistryAuth,
		})
		if err != nil {
			return nil, fmt.Errorf("could not pull %s docker image: %v", opts.StreamImage, err)
		}
	}

	if opts.DataFolder == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not detect working directory path: %v", err)
		}
		opts.DataFolder = path.Join(dir, defaultServerFolder)
	}
	if err := os.MkdirAll(opts.DataFolder, 0700); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("could not create server data folders: %v", err)
	}
	if err := setHidden(opts.DataFolder); err != nil {
		logrus.Warnf("could not mark %s directory as hidden: %v", opts.DataFolder, err)
	}

	if !opts.ServerSecurityEnabled {
		logrus.Warn("Server security is disabled!" +
			"Please don't use this server in prod, or specify `--with-security-check` flag")
	}

	if opts.ServerSecurityEnabled {
		if opts.ServerSecurityPublicKeyPath == "" || opts.ServerSecurityPrivateKeyPath == "" {
			return nil, errors.New(
				"please provide path to server key files or disable `--with-security-check` flag")
		}

		// parse public key data.
		data, err := ioutil.ReadFile(opts.ServerSecurityPublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("could not read %s public key file: %v",
				opts.ServerSecurityPublicKeyPath, err)
		}
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(data)
		if err != nil {
			return nil, fmt.Errorf("could not parse public key data: %v", err)
		}
		opts.publicKey = publicKey

		// parse private key data.
		data, err = ioutil.ReadFile(opts.ServerSecurityPrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("could not read %s private key file: %v",
				opts.ServerSecurityPrivateKeyPath, err)
		}
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(data)
		if err != nil {
			return nil, fmt.Errorf("could not parse private key data: %v", err)
		}
		opts.privateKey = privateKey

		if !opts.privateKey.PublicKey.Equal(opts.publicKey) {
			return nil, errors.New("private key doesn't match public key")
		}
	}

	if opts.BinFolder == "" {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("could not detect working directory path: %v", err)
		}
		opts.BinFolder = dir
	}

	return &opts, nil
}

func defaultServerAddress() string {
	freePort, err := util.FreePort(defaultServerHost)
	if err != nil {
		logrus.Fatalf("could not find any free port: %v", err)
	}

	return fmt.Sprintf("%s:%d", defaultServerHost, freePort)
}
