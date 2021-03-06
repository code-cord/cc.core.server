package models

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/code-cord/cc.core.server/service"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	defaultPageNumber = 1
	defaultPageSize   = 10
)

// GenerateServerTokenRequest represents generate server token request model.
type GenerateServerTokenRequest struct {
	Audience  string    `json:"aud,omitempty"`
	ExpiresAt time.Time `json:"exp,omitempty"`
	IssuedAt  time.Time `json:"iat,omitempty"`
	Issuer    string    `json:"iss,omitempty"`
	NotBefore time.Time `json:"nbf,omitempty"`
	Subject   string    `json:"sub"`
}

// ServerTokenResponse represents server access token response model.
type ServerTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

// StreamListRequest represents stream list request model.
type StreamListRequest struct {
	Term        string
	LaunchModes []service.StreamLaunchMode
	Statuses    []service.StreamStatus
	SortBy      service.StreamSortByField
	SortOrder   service.StreamSortOrder
	PageSize    int
	Page        int
}

// StreamListResponse represents stream list response model.
type StreamListResponse struct {
	Streams  []StreamInfoResponse `json:"streams"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
	Count    int                  `json:"count"`
	HasNext  bool                 `json:"hasNext"`
	Total    int                  `json:"total"`
}

// StreamInfoResponse represents stream info response model.
type StreamInfoResponse struct {
	UUID        string                   `json:"uuid"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	IP          string                   `json:"ip"`
	Port        int                      `json:"port"`
	LaunchMode  service.StreamLaunchMode `json:"launchMode"`
	StartedAt   time.Time                `json:"startedAt"`
	FinishedAt  *time.Time               `json:"finishedAt,omitempty"`
	Status      service.StreamStatus     `json:"status"`
	Join        StreamJoinConfigResponse `json:"join"`
	Host        HostOwnerInfo            `json:"host"`
}

// StreamJoinConfigResponse represents stream join config response model.
type StreamJoinConfigResponse struct {
	JoinPolicy service.JoinPolicy `json:"policy"`
	JoinCode   string             `join:"code"`
}

// Validate validates request model.
func (req *GenerateServerTokenRequest) Validate() error {
	return validation.Errors{
		"sub": validation.Validate(req.Subject,
			validation.Required,
			validation.Length(10, 64),
		),
	}.Filter()
}

// Validate validates request model.
func (req *StreamListRequest) Validate() error {
	modeValidationRules := []validation.Rule{
		validation.In(
			service.StreamLaunchModeDockerContainer,
			service.StreamLaunchModeStandaloneApp,
		),
	}
	for i := range req.LaunchModes {
		err := validation.Errors{
			"mode": validation.Validate(req.LaunchModes[i],
				modeValidationRules...,
			),
		}.Filter()
		if err != nil {
			return err
		}
	}

	statusValidationRules := []validation.Rule{
		validation.In(
			service.StreamStatusFinished,
			service.StreamStatusRunning,
		),
	}
	for i := range req.Statuses {
		err := validation.Errors{
			"status": validation.Validate(req.Statuses[i],
				statusValidationRules...,
			),
		}.Filter()
		if err != nil {
			return err
		}
	}

	return validation.Errors{
		"sortBy": validation.Validate(req.SortBy,
			validation.In(
				service.StreamSortByFieldUUID,
				service.StreamSortByFieldName,
				service.StreamSortByFieldLaunchMode,
				service.StreamSortByFieldStarted,
				service.StreamSortByFieldStatus,
			),
		),
		"sortOrder": validation.Validate(req.SortOrder,
			validation.In(
				service.StreamSortOrderAsc,
				service.StreamSortOrderDesc,
			),
		),
		"pageSize": validation.Validate(req.PageSize,
			validation.Min(1),
		),
		"page": validation.Validate(req.Page,
			validation.Min(1),
		),
	}.Filter()
}

// Build builds request model from URL.
func (req *StreamListRequest) Build(values url.Values) error {
	req.Term = values.Get("term")
	req.SortBy = service.StreamSortByField(values.Get("sortBy"))
	if req.SortBy == "" {
		req.SortBy = service.StreamSortByFieldUUID
	}
	req.SortOrder = service.StreamSortOrder(values.Get("sortOrder"))
	if req.SortOrder == "" {
		req.SortOrder = service.StreamSortOrderAsc
	}

	if pageSize := values.Get("pageSize"); pageSize != "" {
		size, err := strconv.Atoi(pageSize)
		if err != nil {
			return fmt.Errorf("could not parse pageSize param: %v", err)
		}
		req.PageSize = size
	}
	if req.PageSize == 0 {
		req.PageSize = defaultPageSize
	}

	if page := values.Get("page"); page != "" {
		number, err := strconv.Atoi(page)
		if err != nil {
			return fmt.Errorf("could not parse page param: %v", err)
		}
		req.Page = number
	}
	if req.Page == 0 {
		req.Page = defaultPageNumber
	}

	modes := values["mode"]
	req.LaunchModes = make([]service.StreamLaunchMode, len(modes))
	for i := range modes {
		req.LaunchModes[i] = service.StreamLaunchMode(modes[i])
	}

	statuses := values["status"]
	req.Statuses = make([]service.StreamStatus, len(statuses))
	for i := range statuses {
		req.Statuses[i] = service.StreamStatus(statuses[i])
	}

	return nil
}
