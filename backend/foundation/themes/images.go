package themes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

// Image limits: a theme image is decoded, so its size is checked first (decompression bomb)
const (
	MaxImageSide   = 8000
	MaxImagePixels = 20_000_000
	WallpaperWidth = 2560
	PreviewWidth   = 1600
	ThumbWidth     = 480
)

// ErrImage is returned for a file that is not an acceptable image.
var ErrImage = errors.New("not a PNG, JPEG or WebP image of acceptable size")

// Image is a theme image as stored: always re-encoded by us, never the downloaded bytes.
type Image struct {
	ContentType   string
	Data          []byte
	Width, Height int
	SHA256        string
}

// ProcessImage checks a downloaded image and re-encodes it (JPEG, or PNG when it has
// transparency), scaled down to maxWidth. Re-encoding drops whatever else the file carried
// (metadata such as GPS positions, a script or HTML appended to the image), so only pixels
// reach the browser. Only PNG, JPEG and WebP are accepted — never SVG (it can hold scripts)
// or GIF.
func ProcessImage(raw []byte, maxWidth int) (Image, error) {
	if !sniffImage(raw) {
		return Image{}, ErrImage
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") {
		return Image{}, ErrImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > MaxImageSide || cfg.Height > MaxImageSide ||
		cfg.Width*cfg.Height > MaxImagePixels {
		return Image{}, fmt.Errorf("%w (%dx%d)", ErrImage, cfg.Width, cfg.Height)
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return Image{}, ErrImage
	}
	return encode(src, maxWidth)
}

// Thumbnail makes a small JPEG/PNG of an image already processed by ProcessImage.
func Thumbnail(img Image) (Image, error) {
	src, _, err := image.Decode(bytes.NewReader(img.Data))
	if err != nil {
		return Image{}, ErrImage
	}
	return encode(src, ThumbWidth)
}

func encode(src image.Image, maxWidth int) (Image, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > maxWidth {
		h = max(h*maxWidth/w, 1)
		w = maxWidth
		dst := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		src = dst
	}
	var buf bytes.Buffer
	out := Image{Width: w, Height: h}
	if o, ok := src.(interface{ Opaque() bool }); ok && !o.Opaque() {
		if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, src); err != nil {
			return Image{}, err
		}
		out.ContentType = "image/png"
	} else {
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 85}); err != nil {
			return Image{}, err
		}
		out.ContentType = "image/jpeg"
	}
	out.Data = buf.Bytes()
	sum := sha256.Sum256(out.Data)
	out.SHA256 = hex.EncodeToString(sum[:])
	return out, nil
}

// sniffImage checks the magic bytes of PNG, JPEG and WebP.
func sniffImage(b []byte) bool {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return true
	case bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}):
		return true
	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return true
	}
	return false
}
