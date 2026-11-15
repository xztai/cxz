package media

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestInspectPublicationMedia(t *testing.T) {
	for _, kind := range []string{"image", "audio", "video"} {
		if _, _, err := InspectFile(bytes.NewReader([]byte(`<svg onload="alert(1)"></svg>`)), kind); err == nil {
			t.Fatal("active content accepted", kind)
		}
	}
	mime, thumb, err := InspectFile(bytes.NewReader([]byte("<html>download</html>")), "attachment")
	if err != nil || mime != "application/octet-stream" || len(thumb) != 0 {
		t.Fatal("attachment mishandled", err)
	}
	var large bytes.Buffer
	if err := png.Encode(&large, image.NewGray(image.Rect(0, 0, 9000, 1))); err != nil {
		t.Fatal(err)
	}
	if _, _, err := InspectFile(bytes.NewReader(large.Bytes()), "image"); err == nil {
		t.Fatal("oversized image accepted")
	}
}
