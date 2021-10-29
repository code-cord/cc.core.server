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

type responseDecoder func(r io.ReadCloser) error

type requestParams struct {
	basePath              string
	method                string
	queryParams           map[string][]string
	body                  interface{}
	out                   interface{}
	customResponseDecoder responseDecoder
	expStatusCode         int
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
	req := requestParams{
		basePath:      "/",
		method:        http.MethodGet,
		out:           &out,
		expStatusCode: http.StatusOK,
	}
	if err := c.doRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// Ping pings server.
func (c *Client) Ping(ctx context.Context) (*models.PongResponse, error) {
	var out models.PongResponse
	req := requestParams{
		basePath:      "/ping",
		method:        http.MethodGet,
		out:           &out,
		expStatusCode: http.StatusOK,
	}
	if err := c.doRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// NewToken returns new server token.
func (c *Client) NewToken(ctx context.Context, body models.GenerateServerTokenRequest) (
	*models.ServerTokenResponse, error) {
	var out models.ServerTokenResponse
	req := requestParams{
		basePath:      "/token",
		method:        http.MethodPost,
		out:           &out,
		body:          body,
		expStatusCode: http.StatusOK,
	}
	if err := c.doRequest(ctx, req); err != nil {
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
	req := requestParams{
		basePath:      "/stream",
		method:        http.MethodGet,
		queryParams:   queryParams,
		out:           &out,
		expStatusCode: http.StatusOK,
	}
	if err := c.doRequest(ctx, req); err != nil {
		return nil, err
	}

	return &out, nil
}

// FinishStream finishes running stream.
func (c *Client) FinishStream(ctx context.Context, streamUUID string) error {
	req := requestParams{
		basePath:      fmt.Sprintf("/stream/%s", streamUUID),
		method:        http.MethodDelete,
		expStatusCode: http.StatusOK,
	}

	return c.doRequest(ctx, req)
}

// CreateStorageBackup creates storage backup.
func (c *Client) CreateStorageBackup(ctx context.Context, storageName string, w io.Writer) error {
	req := requestParams{
		basePath:      fmt.Sprintf("/storage/%s", storageName),
		method:        http.MethodGet,
		expStatusCode: http.StatusOK,
		customResponseDecoder: func(r io.ReadCloser) error {
			if _, err := io.Copy(w, r); err != nil {
				return fmt.Errorf("could not read response body: %v", err)
			}

			return nil
		},
	}

	return c.doRequest(ctx, req)
}

func (c *Client) doRequest(ctx context.Context, params requestParams) error {
	var bodyReader io.Reader
	if params.body != nil {
		data, err := json.Marshal(params.body)
		if err != nil {
			return fmt.Errorf("could not encode request body: %v", err)
		}

		bodyReader = bytes.NewReader(data)
	}

	reqURL := fmt.Sprintf("%s%s", c.baseAddress, params.basePath)
	req, err := http.NewRequestWithContext(ctx, params.method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("could not create request: %v", err)
	}

	q := req.URL.Query()
	for param, values := range params.queryParams {
		for i := range values {
			q.Add(param, values[i])
		}
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not do request: %v", err)
	}

	if resp.StatusCode != params.expStatusCode {
		var srvErr middleware.Error
		if err := json.NewDecoder(resp.Body).Decode(&srvErr); err != nil {
			return fmt.Errorf("unexpected response: %d %s", resp.StatusCode, resp.Status)
		}

		return srvErr
	}

	if decoder := params.customResponseDecoder; decoder != nil {
		return decoder(resp.Body)
	}

	if params.out == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(params.out); err != nil {
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
