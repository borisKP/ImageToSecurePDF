package main

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"strings"
	"sync/atomic"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type processedPage struct {
	name     string
	jpeg     []byte
	w        int
	h        int
	origW    int
	origH    int
	dpiX     float64
	dpiY     float64
	repaired bool
}

var learnedQuality atomic.Int64

func decodeImage(raw []byte) (image.Image, bool, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err == nil {
		return src, false, nil
	}
	if !isJPEGData(raw) {
		return nil, false, classifyDecodeError(raw, err)
	}
	for _, pad := range []int{64, 256, 1024, 4096} {
		patched := padJPEGTail(raw, pad)
		if patched == nil {
			break
		}
		img, _, e := image.Decode(bytes.NewReader(patched))
		if e == nil {
			return img, true, nil
		}
	}
	return nil, false, classifyDecodeError(raw, err)
}

func isJPEGData(raw []byte) bool {
	return len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 && raw[2] == 0xFF
}

func padJPEGTail(raw []byte, n int) []byte {
	idx := bytes.LastIndex(raw, []byte{0xFF, 0xD9})
	if idx < 0 {
		return nil
	}
	out := make([]byte, 0, len(raw)+n)
	out = append(out, raw[:idx]...)
	for i := 0; i < n; i++ {
		out = append(out, 0x00)
	}
	return append(out, raw[idx:]...)
}

func classifyDecodeError(raw []byte, err error) error {
	msg := err.Error()
	if len(raw) >= 12 && string(raw[4:8]) == "ftyp" {
		brand := string(raw[8:12])
		if strings.HasPrefix(brand, "hei") || strings.HasPrefix(brand, "hev") ||
			brand == "mif1" || brand == "msf1" {
			return E("err.heic", "HEIC/HEIF is not supported; convert to JPG or PNG first (original error: {0})", A(msg)...)
		}
		if strings.HasPrefix(brand, "avi") || brand == "avis" {
			return E("err.avif", "AVIF is not supported; convert to JPG or PNG first (original error: {0})", A(msg)...)
		}
	}
	if isJPEGData(raw) {
		low := strings.ToLower(msg)
		if strings.Contains(low, "missing 0xff00") || strings.Contains(low, "bad huffman") ||
			strings.Contains(low, "unexpected eof") || strings.Contains(low, "truncat") {
			return E("err.jpegTruncated", "Incomplete JPEG data; repair failed (original error: {0})", A(msg)...)
		}
		return E("err.jpegBroken", "Corrupt JPEG or unsupported encoding (original error: {0})", A(msg)...)
	}
	if len(raw) >= 4 && string(raw[:4]) == "RIFF" {
		return E("err.webpBroken", "Corrupt WebP (original error: {0})", A(msg)...)
	}
	return E("err.unknownImage", "Unrecognised or corrupt image (original error: {0})", A(msg)...)
}

func processImage(raw []byte, maxEdge, maxBytes, maxQuality int) (*processedPage, error) {
	return processImageEx(raw, "", maxEdge, maxBytes, maxQuality,
		false, "auto", 96, 150, 0)
}

func processImageEx(raw []byte, name string, maxEdge, maxBytes, quality int,
	lossless bool, dpiSource string, defaultDpi, fixedDpi, maxLongEdgeMm float64) (*processedPage, error) {
	src, repaired, err := decodeImage(raw)
	if err != nil {
		return nil, err
	}
	orient := exifOrientation(raw)
	src = applyOrientation(src, orient)

	b := src.Bounds()
	origW, origH := b.Dx(), b.Dy()
	if origW <= 0 || origH <= 0 {
		return nil, E("err.imageDims", "Invalid image dimensions")
	}
	dpiX, dpiY := readDPI(raw)
	if dpiSource == "fixed" && fixedDpi > 0 {
		dpiX, dpiY = fixedDpi, fixedDpi
	} else if dpiX <= 0 || dpiY <= 0 {
		dpiX, dpiY = defaultDpi, defaultDpi
	}

	if lossless && !repaired && orient == 1 && isJPEGData(raw) {
		return &processedPage{
			name: name, jpeg: raw, w: origW, h: origH,
			origW: origW, origH: origH, dpiX: dpiX, dpiY: dpiY,
		}, nil
	}

	edge := maxEdge
	for attempt := 0; attempt < 4; attempt++ {
		w, h := origW, origH
		nw, nh := w, h
		long := w
		if h > long {
			long = h
		}
		if long > edge {
			s := float64(edge) / float64(long)
			nw = int(float64(w)*s + 0.5)
			nh = int(float64(h)*s + 0.5)
			if nw < 16 {
				nw = 16
			}
			if nh < 16 {
				nh = 16
			}
		}
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
		flattenAlpha(dst)

		data, _, err := encodeTarget(dst, maxBytes, quality)
		if err != nil {
			return nil, err
		}
		out := &processedPage{
			name: name, jpeg: data, w: nw, h: nh,
			origW: origW, origH: origH, dpiX: dpiX, dpiY: dpiY, repaired: repaired,
		}
		if len(data) <= maxBytes || attempt == 3 {
			return out, nil
		}
		edge = edge * 3 / 4
		if edge < 400 {
			edge = 400
		}
	}
	return nil, E("err.encode", "Image encoding failed")
}

func flattenAlpha(dst *image.RGBA) {
	p := dst.Pix
	opaque := true
	for i := 3; i < len(p); i += 4 {
		if p[i] != 0xFF {
			opaque = false
			break
		}
	}
	if opaque {
		return
	}
	for i := 0; i+3 < len(p); i += 4 {
		a := uint32(p[i+3])
		switch a {
		case 0xFF:
			continue
		case 0:
			p[i], p[i+1], p[i+2], p[i+3] = 255, 255, 255, 255
		default:
			inv := 255 - a
			p[i] = uint8((uint32(p[i])*a + 255*inv) / 255)
			p[i+1] = uint8((uint32(p[i+1])*a + 255*inv) / 255)
			p[i+2] = uint8((uint32(p[i+2])*a + 255*inv) / 255)
			p[i+3] = 255
		}
	}
}

func encodeAt(img image.Image, q int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: q}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeTarget(img image.Image, maxBytes, maxQuality int) ([]byte, int, error) {
	if maxQuality > 95 {
		maxQuality = 95
	}
	if maxQuality < 25 {
		maxQuality = 25
	}
	lo, hi := 20, maxQuality
	bestQ := 20
	var best []byte
	if h := int(learnedQuality.Load()); h > 20 && h <= maxQuality {
		if b, e := encodeAt(img, h); e == nil {
			if len(b) <= maxBytes {
				best, bestQ, lo = b, h, h+1
			} else {
				hi = h - 1
			}
		}
	}
	for lo <= hi {
		mid := (lo + hi) / 2
		b, e := encodeAt(img, mid)
		if e != nil {
			return nil, 0, e
		}
		if len(b) <= maxBytes {
			bestQ, best = mid, b
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if best == nil {
		b, e := encodeAt(img, 20)
		if e != nil {
			return nil, 0, e
		}
		learnedQuality.Store(20)
		return b, 20, nil
	}
	learnedQuality.Store(int64(bestQ))
	return best, bestQ, nil
}
