package themes

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/jpeg"
	"testing"
)

func TestProcessImageReencodes(t *testing.T) {
	// A PNG with an HTML/script payload appended (polyglot): only the pixels come out
	poly := append(pngBytes(t, 50, 20), []byte("<html><script>alert(document.cookie)</script></html>")...)
	img, err := ProcessImage(poly, WallpaperWidth)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(img.Data, []byte("<script")) || img.Width != 50 || img.Height != 20 || img.ContentType != "image/jpeg" {
		t.Errorf("polyglot: %s %dx%d, script kept %v", img.ContentType, img.Width, img.Height, bytes.Contains(img.Data, []byte("<script")))
	}
	if len(img.SHA256) != 64 {
		t.Errorf("sha256 %q", img.SHA256)
	}

	// Metadata (EXIF, e.g. GPS) is dropped
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 10, 10)), nil)
	exif := append([]byte{0xff, 0xe1, 0x00, 0x16}, []byte("Exif\x00\x00GPSLatitude=59.9")...)
	withExif := append(append(append([]byte{}, buf.Bytes()[:2]...), exif...), buf.Bytes()[2:]...)
	img, err = ProcessImage(withExif, WallpaperWidth)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(img.Data, []byte("GPSLatitude")) {
		t.Error("EXIF kept")
	}

	// Wide images are scaled down
	img, err = ProcessImage(pngBytes(t, 3000, 30), WallpaperWidth)
	if err != nil || img.Width != WallpaperWidth || img.Height != 25 {
		t.Errorf("scaled: %dx%d, %v", img.Width, img.Height, err)
	}
	thumb, err := Thumbnail(img)
	if err != nil || thumb.Width != ThumbWidth {
		t.Errorf("thumb: %d, %v", thumb.Width, err)
	}
}

// pngHeader is a PNG that only declares its size (IHDR) — enough for DecodeConfig.
func pngHeader(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 2 // 8 bit RGB
	chunk := append([]byte("IHDR"), ihdr...)
	_ = binary.Write(&b, binary.BigEndian, uint32(13))
	b.Write(chunk)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestProcessImageRejects(t *testing.T) {
	cases := map[string][]byte{
		"svg":         []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`),
		"html":        []byte("<!doctype html><script>alert(1)</script>"),
		"gif":         []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"),
		"empty":       nil,
		"pixel bomb":  pngHeader(60000, 60000),
		"too wide":    pngHeader(9000, 10),
		"many pixels": pngHeader(5000, 5000),
		"truncated":   pngBytes(t, 10, 10)[:40],
	}
	for name, data := range cases {
		if _, err := ProcessImage(data, WallpaperWidth); !errors.Is(err, ErrImage) {
			t.Errorf("%s: err = %v, want ErrImage", name, err)
		}
	}
}
