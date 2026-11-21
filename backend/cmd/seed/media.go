package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"etheyl.local/backend/internal/media"
	"etheyl.local/backend/internal/store"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	postPrefix    = "posts/seed-etheyl-v1/"
	profilePrefix = "profiles/seed-etheyl-v1/"
)

func gradient(width, height, hue int, label string, asJPEG bool) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			r := (53 + hue*47 + 90*x/width + 30*y/height) % 256
			g := (96 + hue*37 + 50*x/width + 75*y/height) % 256
			b := (146 + hue*15 + 55*x/width + 45*y/height) % 256
			img.SetRGBA(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	draw.Draw(img, image.Rect(0, height*3/4, width, height), image.NewUniform(color.RGBA{18, 24, 31, 255}), image.Point{}, draw.Src)
	text := image.NewRGBA(image.Rect(0, 0, len(label)*7, 15))
	d := font.Drawer{Dst: text, Src: image.White, Face: basicfont.Face7x13, Dot: fixed.P(0, 12)}
	d.DrawString(label)
	scale := min(4, (width-32)/text.Bounds().Dx(), height/64)
	x, y := width/16, height*7/8-text.Bounds().Dy()*scale/2
	xdraw.NearestNeighbor.Scale(img, image.Rect(x, y, x+text.Bounds().Dx()*scale, y+text.Bounds().Dy()*scale), text, text.Bounds(), draw.Over, nil)
	var out bytes.Buffer
	var err error
	if asJPEG {
		err = jpeg.Encode(&out, img, &jpeg.Options{Quality: 87})
	} else {
		err = png.Encode(&out, img)
	}
	return out.Bytes(), err
}

func wave(seconds, pitch int) []byte {
	const rate = 22050
	n := rate * seconds
	data := []byte("RIFF")
	data = binary.LittleEndian.AppendUint32(data, uint32(36+2*n))
	data = append(data, "WAVEfmt "...)
	data = binary.LittleEndian.AppendUint32(data, 16)
	data = binary.LittleEndian.AppendUint16(data, 1)
	data = binary.LittleEndian.AppendUint16(data, 1)
	data = binary.LittleEndian.AppendUint32(data, rate)
	data = binary.LittleEndian.AppendUint32(data, rate*2)
	data = binary.LittleEndian.AppendUint16(data, 2)
	data = binary.LittleEndian.AppendUint16(data, 16)
	data = append(data, "data"...)
	data = binary.LittleEndian.AppendUint32(data, uint32(2*n))
	for i := range n {
		elapsed := float64(i) / rate
		fade := min(1, elapsed*6, (float64(seconds)-elapsed)*6)
		a := math.Sin(2 * math.Pi * float64(pitch) * elapsed)
		b := math.Sin(3 * math.Pi * float64(pitch) * elapsed)
		sample := int16((a + b/3) * fade * 4700)
		data = binary.LittleEndian.AppendUint16(data, uint16(sample))
	}
	return data
}

func (s *seeder) video(ffmpeg string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "etheyl-seed-video-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	output := filepath.Join(dir, "route.mp4")
	err = exec.CommandContext(s.ctx, ffmpeg, "-nostdin", "-v", "error", "-f", "lavfi",
		"-i", "testsrc2=size=480x270:rate=24", "-t", "4", "-c:v", "libx264",
		"-threads", "1", "-pix_fmt", "yuv420p", "-movflags", "+faststart", output).Run()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(output)
}

func (s *seeder) profileImage(kind string, width, height, hue int, label string) error {
	raw, err := gradient(width, height, hue, label, false)
	if err != nil {
		return err
	}
	img, err := media.Validate(bytes.NewReader(raw), "image/png")
	if err != nil {
		return err
	}
	key := profilePrefix + kind + "-" + img.SHA256
	if err := s.s3.Put(s.ctx, key, img); err != nil {
		return err
	}
	s.keys[key] = true
	_, err = s.tx.Exec(s.ctx, `
		INSERT INTO profile_images(user_id,kind,object_key,content_type,size_bytes,width,height,sha256)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, s.users[alina], kind, key, img.ContentType, len(img.Data), img.Width, img.Height, img.SHA256)
	return err
}

func (s *seeder) file(index int, kind, filename string, raw []byte) error {
	reader := bytes.NewReader(raw)
	contentType, thumbnail, err := media.InspectFile(reader, kind)
	if err != nil {
		return err
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return err
	}
	key := fmt.Sprintf("%s%d/%s-%x", postPrefix, index, filename, sha256.Sum256(raw))
	thumbKey := ""
	if len(thumbnail) > 0 {
		thumbKey = key + "/thumbnail"
	}
	author := s.users[alina]
	row, err := s.q.ReservePostMedia(s.ctx, store.ReservePostMediaParams{
		AuthorID: author, PostID: nullableID(s.posts[index]), Kind: kind, Filename: filename,
		ContentType: contentType, SizeBytes: int64(len(raw)), ObjectKey: key, ThumbnailKey: thumbKey,
	})
	if err != nil {
		return err
	}
	if err := s.s3.Upload(s.ctx, key, reader, int64(len(raw)), contentType); err != nil {
		return err
	}
	s.keys[key] = true
	if thumbKey != "" {
		if err := s.s3.Upload(s.ctx, thumbKey, bytes.NewReader(thumbnail), int64(len(thumbnail)), "image/png"); err != nil {
			return err
		}
		s.keys[thumbKey] = true
	}
	_, err = s.q.CompletePostMedia(s.ctx, store.CompletePostMediaParams{ID: row.ID, AuthorID: author})
	return err
}

func (s *seeder) createMedia(ffmpeg string) error {
	if err := s.profileImage("avatar", 360, 360, 0, "ALINA"); err != nil {
		return err
	}
	if err := s.profileImage("cover", 1440, 480, 1, "CITY LIGHT / STUDIO NOTES"); err != nil {
		return err
	}
	for i, f := range []struct {
		index, width, height int
		name, label          string
		jpeg                 bool
	}{
		{rangePost, 1280, 800, "01-river.jpg", "RIVER WALK / 01", true},
		{rangePost, 1024, 768, "02-perspective.png", "RIVER WALK / 02", false},
		{rangePost, 320, 180, "03-contact-sheet.png", "CONTACT SHEET", false},
		{photoPost, 940, 1200, "01-after-rain.jpg", "AFTER THE RAIN", true},
		{photoPost, 1800, 1000, "02-evening.png", "EVENING LIGHT", false},
		{videoPost, 640, 400, "workshop.jpg", "WORKSHOP", true},
	} {
		raw, err := gradient(f.width, f.height, i+2, f.label, f.jpeg)
		if err != nil {
			return err
		}
		if err := s.file(f.index, "image", f.name, raw); err != nil {
			return err
		}
	}
	for _, f := range []struct{ index, seconds, pitch int }{{rangePost, 3, 220}, {audioPost, 6, 330}, {audioPost, 2, 440}} {
		if err := s.file(f.index, "audio", fmt.Sprintf("field-tone-%d.wav", f.pitch), wave(f.seconds, f.pitch)); err != nil {
			return err
		}
	}
	if ffmpeg != "" {
		raw, err := s.video(ffmpeg)
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		if err != nil {
			fmt.Println("Видео пропущено: ffmpeg не смог создать MP4.")
		} else {
			if err := s.file(videoPost, "video", "evening.mp4", raw); err != nil {
				return err
			}
			if err := s.file(rangePost, "video", "river-walk.mp4", raw); err != nil {
				return err
			}
		}
	}
	if err := s.file(rangePost, "attachment", "route-notes.txt", []byte("Пробный маршрут\n1. Трамвайная остановка\n2. Старый мост\n3. Пристань\n")); err != nil {
		return err
	}
	if err := s.file(photoPost, "attachment", "exposure-notes.txt", []byte("Настройки съёмки\nISO 400, f/5.6, 1/125 с.\n")); err != nil {
		return err
	}
	const size = 50 << 20
	line := []byte("Seed test data for HTTP byte ranges. River route, workshop archive.\n")
	large := bytes.Repeat(line, size/len(line)+1)[:size]
	return s.file(rangePost, "attachment", "route-archive-50mb.txt", large)
}
