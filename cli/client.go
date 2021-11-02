package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/code-cord/cc.core.server/handler/middleware"
	"github.com/code-cord/cc.core.server/handler/models"
)

// Client represents cli http client implementation model.
type Client struct {
	baseAddress string
	httpClient  *http.Client
}

// Option represents set cli client option type.
type Option func(*Options)

// Options represents cli client options model.
type Options struct {
	ServerAddress string
}

// ResponseDecoder represents custom defined response body decoder.
type ResponseDecoder func(r io.ReadCloser) error

// RequestParams represents request params model.
type RequestParams struct {
	Client                *http.Client
	BasePath              string
	BaseAddress           string
	Method                string
	QueryParams           map[string][]string
	Body                  interface{}
	Out                   interface{}
	CustomResponseDecoder ResponseDecoder
	ExpStatusCode         int
}

// NewClient returns new cli client instance.
func NewClient(opt ...Option) Client {
	var opts Options
	for _, o := range opt {
		o(&opts)
	}

	return Client{
		baseAddress: fmt.Sprintf("http://%s", opts.ServerAddress),
		httpClient:  http.DefaultClient,
	}
}

// Info returns server info.
func (c *Client) Info(ctx context.Context) (*models.ServerInfoResponse, error) {
	var out models.ServerInfoResponse
	req := RequestParams{
		Client:        c.httpClient,
		BaseAddress:   c.baseAddress,
		BasePath:      "/",
		Method:        http.MethodGet,
		Out:           &out,
		ExpStatusCode: http.StatusOK,
	}
	if err := DoRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// Ping pings server.
func (c *Client) Ping(ctx context.Context) (*models.PongResponse, error) {
	var out models.PongResponse
	req := RequestParams{
		Client:        c.httpClient,
		BaseAddress:   c.baseAddress,
		BasePath:      "/ping",
		Method:        http.MethodGet,
		Out:           &out,
		ExpStatusCode: http.StatusOK,
	}
	if err := DoRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// NewToken returns new server token.
func (c *Client) NewToken(ctx context.Context, body models.GenerateServerTokenRequest) (
	*models.ServerTokenResponse, error) {
	var out models.ServerTokenResponse
	req := RequestParams{
		Client:        c.httpClient,
		BaseAddress:   c.baseAddress,
		BasePath:      "/token",
		Method:        http.MethodPost,
		Out:           &out,
		Body:          body,
		ExpStatusCode: http.StatusOK,
	}
	if err := DoRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// GetStreams returns streams list.
func (c *Client) GetStreams(ctx context.Context, filter models.StreamListRequest) (
	*models.StreamListResponse, error) {
	queryParams := make(map[string][]string)
	queryParams["term"] = []string{filter.Term}
	modes := make([]string, len(filter.LaunchModes))
	for i := range filter.LaunchModes {
		modes[i] = string(filter.LaunchModes[i])
	}
	queryParams["mode"] = modes
	statuses := make([]string, len(filter.Statuses))
	for i := range filter.Statuses {
		statuses[i] = string(filter.Statuses[i])
	}
	queryParams["status"] = statuses
	queryParams["sortBy"] = []string{string(filter.SortBy)}
	queryParams["sortOrder"] = []string{string(filter.SortOrder)}
	queryParams["pageSize"] = []string{strconv.Itoa(filter.PageSize)}
	queryParams["page"] = []string{strconv.Itoa(filter.Page)}

	var out models.StreamListResponse
	req := RequestParams{
		Client:        c.httpClient,
		BaseAddress:   c.baseAddress,
		BasePath:      "/stream",
		Method:        http.MethodGet,
		QueryParams:   queryParams,
		Out:           &out,
		ExpStatusCode: http.StatusOK,
	}
	if err := DoRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// FinishStream finishes running stream.
func (c *Client) FinishStream(ctx context.Context, streamUUID string) error {
	req := RequestParams{
		Client:        c.httpClient,
		BaseAddress:   c.baseAddress,
		BasePath:      fmt.Sprintf("/stream/%s", streamUUID),
		Method:        http.MethodDelete,
		ExpStatusCode: http.StatusOK,
	}

	return DoRequest(ctx, req)
}

// CreateStorageBackup creates storage backup.
func (c *Client) CreateStorageBackup(ctx context.Context, storageName string, w io.Writer) error {
	req := RequestParams{
		Client:        c.httpClient,
		BaseAddress:   c.baseAddress,
		BasePath:      fmt.Sprintf("/storage/%s", storageName),
		Method:        http.MethodGet,
		ExpStatusCode: http.StatusOK,
		CustomResponseDecoder: func(r io.ReadCloser) error {
			if _, err := io.Copy(w, r); err != nil {
				return fmt.Errorf("could not read response body: %v", err)
			}

			return nil
		},
	}

	return DoRequest(ctx, req)
}

// DoRequest sends http request with provided configuration.
func DoRequest(ctx context.Context, params RequestParams) error {
	var bodyReader io.Reader
	if params.Body != nil {
		data, err := json.Marshal(params.Body)
		if err != nil {
			return fmt.Errorf("could not encode request body: %v", err)
		}

		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", params.BaseAddress, params.BasePath)
	req, err := http.NewRequestWithContext(ctx, params.Method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("could not create request: %v", err)
	}

	q := req.URL.Query()
	for param, values := range params.QueryParams {
		for i := range values {
			q.Add(param, values[i])
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := params.Client.Do(req)
	if err != nil {
		return fmt.Errorf("could not do request: %v", err)
	}

	if resp.StatusCode != params.ExpStatusCode {
		var srvErr middleware.Error
		if err := json.NewDecoder(resp.Body).Decode(&srvErr); err != nil {
			return fmt.Errorf("unexpected response: %d %s", resp.StatusCode, resp.Status)
		}

		return srvErr
	}

	if decoder := params.CustomResponseDecoder; decoder != nil {
		return decoder(resp.Body)
	}

	if params.Out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(params.Out); err != nil {
		return fmt.Errorf("could not parse response body: %v", err)
	}

	return nil
}

// Address sets server API address option.
func Address(address string) Option {
	return func(o *Options) {
		o.ServerAddress = address
	}
}
