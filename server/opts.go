package server

import (
	"crypto/rsa"

	"github.com/sirupsen/logrus"
)

// Option represents set server option type.
type Option func(*Options)

// Options represents server options model.
type Options struct {
	Name                         string
	Description                  string
	Version                      string
	Meta                         map[string]interface{}
	Address                      string
	TLSCertFile                  string
	TLSKeyFile                   string
	LogLevel                     string
	StreamContainerPrefix        string
	StreamImage                  string
	StreamImageRegistryAuth      string
	PullImageOnStartup           bool
	DataFolder                   string
	MaxAvatarSize                int64
	ServerSecurityPublicKeyPath  string
	ServerSecurityPrivateKeyPath string
	ServerSecurityEnabled        bool
	BinFolder                    string

	logLevel   logrus.Level
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
	tlsEnabled bool
}

// Name sets server name option.
func Name(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

// Description sets server description option.
func Description(desc string) Option {
	return func(o *Options) {
		o.Description = desc
	}
}

// Version sets server version option.
func Version(ver string) Option {
	return func(o *Options) {
		o.Version = ver
	}
}

// Address sets server serve address option.
func Address(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}

// TLS sets server TLS option.
func TLS(certFile, keyFile string) Option {
	return func(o *Options) {
		o.TLSCertFile = certFile
		o.TLSKeyFile = keyFile
	}
}

// LogLevel sets server log level option.
//
// Possible values are:
// - "panic"
// - "fatal"
// - "error"
// - "warn" ("warning")
// - "info"
// - "debug"
// - "trace"
func LogLevel(level string) Option {
	return func(o *Options) {
		o.LogLevel = level
	}
}

// Meta sets server meta option.
func Meta(meta map[string]interface{}) Option {
	return func(o *Options) {
		o.Meta = meta
	}
}

// StreamContainerPrefix sets stream container prefix for streams running inside docker containers.
func StreamContainerPrefix(prefix string) Option {
	return func(o *Options) {
		o.StreamContainerPrefix = prefix
	}
}

// DataFolder sets folder to store server data.
func DataFolder(folder string) Option {
	return func(o *Options) {
		o.DataFolder = folder
	}
}

// MaxAvatarSize sets maximum avatar size in bytes.
func MaxAvatarSize(size int64) Option {
	return func(o *Options) {
		o.MaxAvatarSize = size
	}
}

// BinFolder sets path to the code-cord binaries.
func BinFolder(folder string) Option {
	return func(o *Options) {
		o.BinFolder = folder
	}
}

// StreamImage sets stream docker image name.
func StreamImage(img string) Option {
	return func(o *Options) {
		o.StreamImage = img
	}
}

// ServerPublicKey sets server RSA public security key.
func ServerPublicKey(keyPath string) Option {
	return func(o *Options) {
		o.ServerSecurityPublicKeyPath = keyPath
	}
}

// ServerPrivateKey sets server RSA private security key.
func ServerPrivateKey(keyPath string) Option {
	return func(o *Options) {
		o.ServerSecurityPrivateKeyPath = keyPath
	}
}

// ServerSecurityEnabled sets server security state.
func ServerSecurityEnabled(enabled bool) Option {
	return func(o *Options) {
		o.ServerSecurityEnabled = enabled
	}
}

// PullImageOnStartup enable or disable stream docker image pulling on server startup.
func PullImageOnStartup(pull bool) Option {
	return func(o *Options) {
		o.PullImageOnStartup = pull
	}
}

// StreamImageRegistryAuth is the base64 encoded credentials for the pulling stream image.
func StreamImageRegistryAuth(auth string) Option {
	return func(o *Options) {
		o.StreamImageRegistryAuth = auth
	}
}
