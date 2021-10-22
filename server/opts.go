package server

import "github.com/sirupsen/logrus"

// Option represents set server option type.
type Option func(*Options)

// Options represents server options model.
type Options struct {
	Name        string
	Description string
	Version     string
	Address     string
	TLSCertFile string
	TLSKeyFile  string
	LogLevel    string
	logLevel    logrus.Level
	Meta        map[string]interface{}
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
