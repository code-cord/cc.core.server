package util

import (
	"fmt"
	"net"
	"net/http"
)

// FreePort returns free system open port that is ready to use.
func FreePort(host string) (int, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", host))
	if err != nil {
		return 0, err
	}

	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return 0, err
	}

	return tcpListener.Addr().(*net.TCPAddr).Port, tcpListener.Close()
}

// GetIP returns IP address of the request.
func GetIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-FORWARDED-FOR"); forwarded != "" {
		return forwarded
	}

	return r.RemoteAddr
}
