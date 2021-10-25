package api

import (
	"context"
	"io"
)

// Avatar describes avatar service API.
type Avatar interface {
	New(ctx context.Context, contentType string, r io.Reader) (string, error)
	Restrictions() AvatarRestrictions
	ByID(ctx context.Context, avatarID string) (imgData []byte, contentType string, err error)
}

// AvatarRestrictions represents avatar restrictions model.
type AvatarRestrictions struct {
	MaxFileSize int64
}
