package media

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	"image/png"
	"io"
	"net/http"
	"strings"

	"golang.org/x/image/draw"
)

var ErrFile = errors.New("invalid media file")

// InspectFile preserves the original. Only the separate image thumbnail is
// encoded. SVG and HTML are accepted exclusively as downloadable attachments.
func InspectFile(file io.ReadSeeker, kind string) (string, []byte, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", nil, err
	}
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", nil, err
	}
	detected := http.DetectContentType(head[:n])
	switch kind {
	case "attachment":
		return "application/octet-stream", nil, nil
	case "audio":
		switch detected {
		case "audio/mpeg", "audio/ogg", "audio/wave", "audio/midi", "audio/aiff", "audio/webm":
			return detected, nil, nil
		case "application/ogg":
			return "audio/ogg", nil, nil
		case "video/webm":
			return "audio/webm", nil, nil
		case "video/mp4":
			return "audio/mp4", nil, nil
		}
		if bytes.HasPrefix(head[:n], []byte("fLaC")) {
			return "audio/flac", nil, nil
		}
	case "video":
		if detected == "video/mp4" || detected == "video/webm" || detected == "video/ogg" {
			return detected, nil, nil
		}
		if detected == "application/ogg" {
			return "video/ogg", nil, nil
		}
	case "image":
		cfg, format, err := image.DecodeConfig(file)
		if err != nil || !strings.Contains("|png|jpeg|gif|webp|", "|"+format+"|") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 16000000 {
			return "", nil, ErrFile
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", nil, err
		}
		img, _, err := image.Decode(file)
		if err != nil {
			return "", nil, ErrFile
		}
		width, height := cfg.Width, cfg.Height
		if max(width, height) > 640 {
			width = max(1, width*640/max(cfg.Width, cfg.Height))
			height = max(1, height*640/max(cfg.Width, cfg.Height))
		}
		thumb := image.NewNRGBA(image.Rect(0, 0, width, height))
		draw.CatmullRom.Scale(thumb, thumb.Bounds(), img, img.Bounds(), draw.Src, nil)
		var out bytes.Buffer
		if err := png.Encode(&out, thumb); err != nil {
			return "", nil, err
		}
		return map[string]string{"png": "image/png", "jpeg": "image/jpeg", "gif": "image/gif", "webp": "image/webp"}[format], out.Bytes(), nil
	}
	return "", nil, ErrFile
}
