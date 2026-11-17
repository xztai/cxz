package media

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	_ "golang.org/x/image/webp"
)

const MaxBytes = 5 << 20

var ErrImage = errors.New("invalid image")

type Image struct {
	Data          []byte
	ContentType   string
	Width, Height int
	SHA256        string
}

// Validate fully decodes and re-encodes images, discarding metadata and trailing payloads.
// Dimensions are bounded before allocation, regardless of the declared MIME or filename.
func Validate(r io.Reader, declared string) (Image, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Image{}, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 || len(data) > MaxBytes {
		return Image{}, ErrImage
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	mime := map[string]string{"png": "image/png", "jpeg": "image/jpeg", "webp": "image/webp"}[format]
	if err != nil || mime == "" || declared != mime || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
		return Image{}, ErrImage
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Image{}, ErrImage
	}
	var out bytes.Buffer
	contentType := "image/png"
	if format == "jpeg" {
		contentType = "image/jpeg"
		err = jpeg.Encode(&out, decoded, &jpeg.Options{Quality: 90})
	} else {
		err = png.Encode(&out, decoded)
	}
	if err != nil {
		return Image{}, fmt.Errorf("encode image: %w", err)
	}
	if out.Len() > MaxBytes {
		return Image{}, ErrImage
	}
	sum := sha256.Sum256(out.Bytes())
	return Image{Data: out.Bytes(), ContentType: contentType, Width: cfg.Width, Height: cfg.Height, SHA256: hex.EncodeToString(sum[:])}, nil
}
