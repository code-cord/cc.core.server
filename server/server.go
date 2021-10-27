package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io/ioutil"
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
	apiHandler "github.com/code-cord/cc.core.server/handler/api"
	"github.com/code-cord/cc.core.server/handler/models"
	"github.com/code-cord/cc.core.server/storage"
	"github.com/code-cord/cc.core.server/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultServerHost                  = "127.0.0.1"
	defaultAPIServerHost               = "127.0.0.1"
	defaultAPIServerPort               = 7070
	defaultServerFolder                = ".__data"
	defaultConnectToStreamRetryCount   = 3
	defaultConnectToStreamRetryTimeout = 2 * time.Second
	defaultStreamTokenType             = "bearer"
	defaultAvatarsFolderName           = "avatars"
	defaultStreamStorageName           = "stream.db"
	streamBucket                       = "stream"
	defaultAvatarStorageName           = "avatar.db"
)

// Server represents code-cord server implementation model.
type Server struct {
	opts          Options
	httpServer    *http.Server
	apiHttpServer *http.Server
	streams       *sync.Map
	storage       *storage.Storage
}

type streamInfo struct {
	api.Stream
	name                   string
	description            string
	ip                     string
	port                   int
	join                   api.StreamJoinPolicyConfig
	rcpClient              *rpc.Client
	startedAt              time.Time
	hostInfo               api.HostInfo
	launchMode             api.StreamLaunchMode
	participants           *sync.Map
	pendingParticipantsMap *sync.Map
	rsaKeys                *rsaKeys
	subject                string
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

	db, err := storage.New(storage.Config{
		DBPath:        path.Join(opts.DataFolder, defaultStreamStorageName),
		Buckets:       []string{streamBucket},
		DefaultBucket: streamBucket,
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to storage: %v", err)
	}

	/////
	mm := []models.HostOwnerInfo{
		{
			Username: "1",
			IP:       "1",
		},
		{
			Username: "2",
			IP:       "2",
		},
		{
			Username: "3",
			IP:       "3",
		},
	}
	for i := range mm {
		err := db.Default().Insert(mm[i].Username, mm[i])
		if err != nil {
			panic(err)
		}
	}

	var mm3 []models.HostOwnerInfo
	if err := db.Default().All(&mm3); err != nil {
		panic(err)
	}
	////

	s := Server{
		opts: *opts,
		httpServer: &http.Server{
			Addr: opts.Address,
		},
		apiHttpServer: &http.Server{
			Addr: fmt.Sprintf("%s:%d", defaultAPIServerHost, defaultAPIServerPort),
		},
		streams: new(sync.Map),
		storage: db,
	}
	if opts.LogLevel != "" {
		logrus.SetLevel(opts.logLevel)
	}
	s.httpServer.Handler = handler.New(handler.Config{
		Server:               &s,
		SeverSecurityEnabled: s.opts.ServerSecurityEnabled,
		ServerSecurityKey:    s.opts.ssKey,
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

	if err := s.apiHttpServer.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Sprintf("could not stop API http server: %v", err))
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

// JoinParticipant joins a new particiant to the stream.
func (s *Server) JoinParticipant(
	joinCodectx context.Context, streamUUID, joinCode string, p api.Participant) (
	joinDesicion *api.JoinParticipantDecision, joinErr error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}

	streamData := streamValue.(streamInfo)
	p.UUID = uuid.New().String()
	p.Status = api.ParticipantStatusPending
	streamData.participants.Store(p.UUID, p)

	joinDesicion = new(api.JoinParticipantDecision)
	defer func() {
		if !joinDesicion.JoinAllowed {
			streamData.participants.Delete(p.UUID)
			return
		}

		accessToken, err := generateStreamAccessToken(
			streamUUID, p.UUID, false, streamData.rsaKeys.privateKey)
		if err != nil {
			joinErr = fmt.Errorf("could not generate access token: %v", err)
			return
		}

		joinDesicion.AccessToken = accessToken
		p.Status = api.ParticipantStatusActive
		streamData.participants.Store(p.UUID, p)
	}()

	switch streamData.join.JoinPolicy {
	case api.JoinPolicyAuto:
		joinDesicion.JoinAllowed = true
	case api.JoinPolicyByCode:
		if !strings.EqualFold(streamData.join.JoinCode, joinCode) {
			joinErr = errors.New("invalid join code")
			return
		}
		joinDesicion.JoinAllowed = true
	case api.JoinPolicyHostResolve:
		pendingChan := make(chan bool)
		streamData.pendingParticipantsMap.Store(p.UUID, pendingChan)
		joinDesicion.JoinAllowed = <-pendingChan
	default:
		joinErr = errors.New("unknown stream join policy")
	}

	return
}

// DecideParticipantJoin allows or denies participant to join the stream.
func (s *Server) DecideParticipantJoin(
	ctx context.Context, streamUUID, participantUUID string, joinAllowed bool) error {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}
	streamData := streamValue.(streamInfo)

	participantValue, ok := streamData.pendingParticipantsMap.Load(participantUUID)
	if !ok {
		return fmt.Errorf("could not find pending participant with UUID %s", participantUUID)
	}
	participantChan := participantValue.(chan bool)
	participantChan <- joinAllowed

	return nil
}

// StreamParticipants returns list of stream participants.
func (s *Server) StreamParticipants(ctx context.Context, streamUUID string) (
	[]api.Participant, error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}
	streamData := streamValue.(streamInfo)

	participants := make([]api.Participant, 0)
	streamData.participants.Range(func(key, value interface{}) bool {
		if participant, ok := value.(api.Participant); ok {
			participants = append(participants, participant)
		}

		return true
	})

	return participants, nil
}

// NewStreamHostToken generates new access token for the host of the stream.
func (s *Server) NewStreamHostToken(ctx context.Context, streamUUID, subject string) (
	*api.StreamAuthInfo, error) {
	streamValue, ok := s.streams.Load(streamUUID)
	if !ok {
		return nil, fmt.Errorf("could not find stream with UUID %s", streamUUID)
	}
	streamData := streamValue.(streamInfo)

	if streamData.subject == "" || streamData.subject != subject {
		return nil, errors.New("could not verify stream subject")
	}

	token, err := generateStreamAccessToken(
		streamUUID, streamData.hostInfo.UUID, true, streamData.rsaKeys.privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not generate access token: %v", err)
	}

	return &api.StreamAuthInfo{
		AccessToken: token,
		Type:        defaultStreamTokenType,
	}, nil
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

func buildStreamOwnerInfo(info *streamInfo, streamUUID, accessToken string) *api.StreamOwnerInfo {
	ownerInfo := api.StreamOwnerInfo{
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

	if accessToken != "" {
		ownerInfo.Auth = &api.StreamAuthInfo{
			AccessToken: accessToken,
			Type:        defaultStreamTokenType,
		}
	}

	return &ownerInfo
}

func newServerOptions(opt ...Option) (*Options, error) {
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
			return nil, fmt.Errorf("could not detect working directory path: %v", err)
		}
		opts.DataFolder = path.Join(dir, defaultServerFolder)
		if err := os.MkdirAll(opts.DataFolder, 0700); err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("could not create server data folders: %v", err)
		}
	}

	avatarsFolder := path.Join(opts.DataFolder, defaultAvatarsFolderName)
	if err := os.MkdirAll(avatarsFolder, 0700); err != nil && err != os.ErrExist {
		return nil, fmt.Errorf("could not access the avatars folder: %v", err)
	}
	opts.avatarsFolder = avatarsFolder

	if !opts.ServerSecurityEnabled {
		logrus.Warn("Server security is disabled!" +
			"Please don't use this server in prod, or specify `--with-security-check` flag")
	}

	if opts.ServerSecurityEnabled {
		if opts.ServerSecurityKeyPath == "" {
			return nil, errors.New(
				"please provide path to security key file or disable `--with-security-check` flag")
		}
		data, err := ioutil.ReadFile(opts.ServerSecurityKeyPath)
		if err != nil {
			return nil, fmt.Errorf("could not read %s key file: %v", opts.ServerSecurityKeyPath, err)
		}
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM(data)
		if err != nil {
			return nil, fmt.Errorf("could not parse public key data: %v", err)
		}
		opts.ssKey = publicKey
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

func generateRSAKeys() (*rsaKeys, error) {
	privatekey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("could not generate private RSA key: %v", err)
	}

	return &rsaKeys{
		privateKey: privatekey,
		publicKey:  &privatekey.PublicKey,
	}, nil
}

func generateStreamAccessToken(
	streamUUID, participantUUID string, isHost bool, privateKey *rsa.PrivateKey) (string, error) {
	claims := &jwt.MapClaims{
		"streamUUID": streamUUID,
		"UUID":       participantUUID,
		"host":       isHost,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
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
