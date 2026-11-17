package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestValidateImage(t *testing.T) {
	var data bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 8, 4))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	raw := append(bytes.Clone(data.Bytes()), []byte("<script>trailing payload</script>")...)
	good, err := Validate(bytes.NewReader(raw), "image/png")
	if err != nil || good.Width != 8 || good.Height != 4 || good.ContentType != "image/png" || len(good.SHA256) != 64 || bytes.Contains(good.Data, []byte("script")) {
		t.Fatalf("validation: %+v %v", good, err)
	}
	bomb := bytes.Clone(data.Bytes())
	binary.BigEndian.PutUint32(bomb[16:20], 8000)
	binary.BigEndian.PutUint32(bomb[20:24], 8000)
	binary.BigEndian.PutUint32(bomb[29:33], crc32.ChecksumIEEE(bomb[12:29]))
	cases := []struct {
		name, mime string
		data       []byte
	}{
		{"wrong mime", "image/jpeg", data.Bytes()},
		{"svg", "image/svg+xml", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{"truncated", "image/png", data.Bytes()[:40]},
		{"empty", "image/png", nil},
		{"oversize", "image/png", make([]byte, MaxBytes+1)},
		{"pixel bomb", "image/png", bomb},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Validate(bytes.NewReader(tc.data), tc.mime); err == nil {
				t.Fatal("invalid image accepted")
			}
		})
	}
}
