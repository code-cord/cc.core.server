package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"

	"github.com/code-cord/cc.core.server/service"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
)

const (
	defaultAvatarImageSize = 256
	pngContentType         = "image/png"
	jpegContentType        = "image/jpeg"
)

var imgProcessors map[string]imageProcessor = map[string]imageProcessor{
	pngContentType:  &pngProcessor{},
	jpegContentType: &jpegProcessor{},
}

type imageProcessor interface {
	Decode(r io.Reader) (image.Image, error)
	Encode(img image.Image) ([]byte, error)
}

type avatar struct {
	UUID        string `json:"uuid"`
	ImageData   []byte `json:"img"`
	ContentType string `json:"ct"`
}

type pngProcessor struct{}

type jpegProcessor struct{}

// NewAvatar stores a new avatar image.
func (s *Server) NewAvatar(ctx context.Context, contentType string, r io.Reader) (string, error) {
	imgPcocessor, ok := imgProcessors[contentType]
	if !ok {
		return "", fmt.Errorf("unsupported image type: %s", contentType)
	}

	img, err := imgPcocessor.Decode(r)
	if err != nil {
		return "", fmt.Errorf("could not decode image data: %v", err)
	}

	img = resize.Resize(defaultAvatarImageSize, defaultAvatarImageSize, img, resize.Lanczos3)

	imgData, err := imgPcocessor.Encode(img)
	if err != nil {
		return "", fmt.Errorf("could not encode image data: %v", err)
	}

	avatarID := uuid.New().String()
	if err := s.avatarStorage.Default().Store(avatarID, avatar{
		UUID:        avatarID,
		ImageData:   imgData,
		ContentType: contentType,
	}, json.Marshal); err != nil {
		return "", fmt.Errorf("could not store image: %v", err)
	}

	return avatarID, nil
}

// AvatarRestrictions returns restrictions for the avatar image.
func (s *Server) AvatarRestrictions() service.AvatarRestrictions {
	return service.AvatarRestrictions{
		MaxFileSize: s.opts.MaxAvatarSize,
	}
}

// ByID returns image data by ID.
func (s *Server) AvatarByID(ctx context.Context, avatarID string) (
	imgData []byte, contentType string, err error) {
	rv := s.avatarStorage.Default().Load(avatarID)
	if rv == nil {
		err = os.ErrNotExist
		return
	}

	var a avatar
	if err = rv.Decode(&a, json.Unmarshal); err != nil {
		err = fmt.Errorf("could not read avatar data: %v", err)
		return
	}

	imgData = a.ImageData
	contentType = a.ContentType

	return
}

// Decode decodes png image.
func (p *pngProcessor) Decode(r io.Reader) (image.Image, error) {
	return png.Decode(r)
}

// Encode encodes png image.
func (p *pngProcessor) Encode(img image.Image) ([]byte, error) {
	var b bytes.Buffer

	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Decode decodes jpeg image.
func (p *jpegProcessor) Decode(r io.Reader) (image.Image, error) {
	return jpeg.Decode(r)
}

// Encode encodes jpeg image.
func (p *jpegProcessor) Encode(img image.Image) ([]byte, error) {
	var b bytes.Buffer

	if err := jpeg.Encode(&b, img, nil); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
