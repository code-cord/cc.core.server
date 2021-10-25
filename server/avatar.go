package server

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/code-cord/cc.core.server/api"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

const (
	avatarsFolder          = "avatars"
	defaultAvatarImageSize = 256
	pngImageExtension      = "png"
	jpegImageExtension     = "jpg"
	pngContentType         = "image/png"
	jpegContentType        = "image/jpeg"
)

// AvatarService represents avatar service implementation model.
type AvatarService struct {
	destFolder  string
	maxFileSize int64
}

// AvatarServiceConfig represents avatar service configuration model.
type AvatarServiceConfig struct {
	DataFolder  string
	MaxFileSize int64
}

// NewAvatarService returns new avatar service instance.
func NewAvatarService(cfg AvatarServiceConfig) (*AvatarService, error) {
	fullPath := path.Join(cfg.DataFolder, avatarsFolder)
	if err := os.MkdirAll(fullPath, 0700); err != nil && err != os.ErrExist {
		return nil, err
	}

	s := AvatarService{
		destFolder:  fullPath,
		maxFileSize: cfg.MaxFileSize,
	}

	return &s, nil
}

// New stores a new avatar image.
func (a *AvatarService) New(ctx context.Context, contentType string, r io.Reader) (string, error) {
	var (
		img     image.Image
		err     error
		fileExt string
	)

	switch contentType {
	case pngContentType:
		img, err = png.Decode(r)
		fileExt = pngImageExtension
	case jpegContentType:
		img, err = jpeg.Decode(r)
		fileExt = jpegImageExtension
	default:
		err = fmt.Errorf("unsupported image type: %s", contentType)
	}
	if err != nil {
		return "", err
	}

	// resize image data.
	img = resize.Resize(defaultAvatarImageSize, defaultAvatarImageSize, img, resize.Lanczos3)

	avatarID := uuid.New().String()
	avatarPath := path.Join(a.destFolder, fmt.Sprintf("%s.%s", avatarID, fileExt))
	out, err := os.Create(avatarPath)
	if err != nil {
		return "", fmt.Errorf("could not create image: %v", err)
	}
	defer out.Close()

	switch fileExt {
	case pngImageExtension:
		err = png.Encode(out, img)
	case jpegImageExtension:
		err = jpeg.Encode(out, img, nil)
	}
	if err != nil {
		return "", fmt.Errorf("could not encode image data: %v", err)
	}

	return avatarID, nil
}

// Restrictions returns restrictions for the avatar image.
func (a *AvatarService) Restrictions() api.AvatarRestrictions {
	return api.AvatarRestrictions{
		MaxFileSize: a.maxFileSize,
	}
}

// ByID returns image data by ID.
func (a *AvatarService) ByID(ctx context.Context, avatarID string) (
	imgData []byte, contentType string, err error) {
	avatarPath := path.Join(a.destFolder, avatarID)

	matches, err := filepath.Glob(fmt.Sprintf("%s.*", avatarPath))
	if err != nil {
		return nil, "", err
	}

	if len(matches) == 0 {
		return nil, "", os.ErrNotExist
	}

	avatarPath = matches[0]
	switch path.Ext(avatarPath) {
	case pngImageExtension:
		contentType = pngContentType
	case jpegImageExtension:
		contentType = jpegContentType
	}

	imgData, err = ioutil.ReadFile(avatarPath)

	return
}
