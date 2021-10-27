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

// NewAvatar stores a new avatar image.
func (s *Server) NewAvatar(ctx context.Context, contentType string, r io.Reader) (string, error) {
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
	avatarPath := path.Join(s.opts.avatarsFolder, fmt.Sprintf("%s.%s", avatarID, fileExt))
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

// AvatarRestrictions returns restrictions for the avatar image.
func (s *Server) AvatarRestrictions() api.AvatarRestrictions {
	return api.AvatarRestrictions{
		MaxFileSize: s.opts.MaxAvatarSize,
	}
}

// ByID returns image data by ID.
func (s *Server) AvatarByID(ctx context.Context, avatarID string) (
	imgData []byte, contentType string, err error) {
	avatarPath := path.Join(s.opts.avatarsFolder, avatarID)

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
