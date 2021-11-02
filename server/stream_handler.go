package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/code-cord/cc.core.server/cli"
	"github.com/code-cord/cc.core.server/service"
)

// StreamHandler represents stream handler implementation mnodel.
type StreamHandler struct {
	streamAddress string
	httpClient    *http.Client
}

// NewStreamHandler returns new stream handler instance.
func NewStreamHandler(serveAddress string) *StreamHandler {
	return &StreamHandler{
		httpClient:    http.DefaultClient,
		streamAddress: fmt.Sprintf("http://%s", serveAddress),
	}
}

// NewParticipant reports stream about new participant.
func (h *StreamHandler) NewParticipant(p service.StreamParticipant) error {
	return cli.DoRequest(context.Background(), cli.RequestParams{
		Client:        h.httpClient,
		BasePath:      "/participant",
		BaseAddress:   h.streamAddress,
		Method:        http.MethodPost,
		Body:          p,
		ExpStatusCode: http.StatusOK,
	})
}

// ChangeParticipantInfo reports stream about changing participant info.
func (h *StreamHandler) ChangeParticipantInfo(p service.StreamParticipant) error {
	return cli.DoRequest(context.Background(), cli.RequestParams{
		Client:        h.httpClient,
		BasePath:      "/participant",
		BaseAddress:   h.streamAddress,
		Method:        http.MethodPatch,
		Body:          p,
		ExpStatusCode: http.StatusOK,
	})
}
