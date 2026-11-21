package main

import (
	"encoding/binary"
	"image"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".jfif": true, ".png": true, ".gif": true,
	".bmp": true, ".tif": true, ".tiff": true, ".webp": true,
}

func isImageName(name string) bool {
	return imageExts[strings.ToLower(filepath.Ext(name))]
}

func naturalLess(a, b string) bool {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		ca, cb := rune(a[ai]), rune(b[bi])
		if unicode.IsDigit(ca) && unicode.IsDigit(cb) {
			as, bs := ai, bi
			for ai < len(a) && unicode.IsDigit(rune(a[ai])) {
				ai++
			}
			for bi < len(b) && unicode.IsDigit(rune(b[bi])) {
				bi++
			}
			av, _ := strconv.ParseUint(strings.TrimLeft(a[as:ai], "0"), 10, 64)
			bv, _ := strconv.ParseUint(strings.TrimLeft(b[bs:bi], "0"), 10, 64)
			la, lb := ai-as, bi-bs
			if av != bv {
				return av < bv
			}
			if la != lb {
				return la < lb
			}
			continue
		}
		ua, ub := unicode.ToLower(ca), unicode.ToLower(cb)
		if ua != ub {
			return ua < ub
		}
		ai++
		bi++
	}
	return len(a)-ai < len(b)-bi
}

// readDPI extracts JFIF density or PNG pHYs. Returns 96/96 when absent.
func readDPI(raw []byte) (float64, float64) {
	if len(raw) >= 4 && raw[0] == 0xFF && raw[1] == 0xD8 {
		pos := 2
		for pos+4 < len(raw) && pos < 131072 {
			if raw[pos] != 0xFF {
				break
			}
			marker := raw[pos+1]
			if marker == 0xDA || marker == 0xD9 {
				break
			}
			if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
				pos += 2
				continue
			}
			segLen := int(binary.BigEndian.Uint16(raw[pos+2 : pos+4]))
			if segLen < 2 || pos+2+segLen > len(raw) {
				break
			}
			if marker == 0xE0 {
				seg := raw[pos+4 : pos+2+segLen]
				if len(seg) >= 12 && string(seg[:5]) == "JFIF\x00" {
					units := seg[7]
					xd := float64(binary.BigEndian.Uint16(seg[8:10]))
					yd := float64(binary.BigEndian.Uint16(seg[10:12]))
					switch units {
					case 1:
						return validDPI(xd, yd)
					case 2:
						return validDPI(xd*2.54, yd*2.54)
					}
				}
			}
			pos += 2 + segLen
		}
	}
	if len(raw) >= 8 && string(raw[1:4]) == "PNG" {
		pos := 8
		for pos+12 <= len(raw) {
			n := int(binary.BigEndian.Uint32(raw[pos : pos+4]))
			if n < 0 || pos+12+n > len(raw) {
				break
			}
			typ := string(raw[pos+4 : pos+8])
			if typ == "pHYs" && n >= 9 {
				x := float64(binary.BigEndian.Uint32(raw[pos+8 : pos+12]))
				y := float64(binary.BigEndian.Uint32(raw[pos+12 : pos+16]))
				if raw[pos+16] == 1 {
					return validDPI(x*0.0254, y*0.0254)
				}
			}
			pos += 12 + n
			if typ == "IEND" {
				break
			}
		}
	}
	return 96, 96
}

func validDPI(x, y float64) (float64, float64) {
	if x < 1 || x > 10000 || y < 1 || y > 10000 {
		return 96, 96
	}
	return x, y
}

func exifOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 < len(data) && pos < 131072 {
		if data[pos] != 0xFF {
			break
		}
		marker := data[pos+1]
		if marker == 0xDA || marker == 0xD9 {
			break
		}
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			pos += 2
			continue
		}
		segLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		if segLen < 2 || pos+2+segLen > len(data) {
			break
		}
		seg := data[pos+4 : pos+2+segLen]
		if marker == 0xE1 && len(seg) >= 8 && string(seg[:6]) == "Exif\x00\x00" {
			return parseTiffOrientation(seg[6:])
		}
		pos += 2 + segLen
	}
	return 1
}

func parseTiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder = binary.BigEndian
	switch {
	case t[0] == 'I' && t[1] == 'I':
		bo = binary.LittleEndian
	case t[0] == 'M' && t[1] == 'M':
	default:
		return 1
	}
	if bo.Uint16(t[2:4]) != 42 {
		return 1
	}
	off := int(bo.Uint32(t[4:8]))
	if off+2 > len(t) {
		return 1
	}
	count := int(bo.Uint16(t[off : off+2]))
	for i := 0; i < count; i++ {
		e := off + 2 + i*12
		if e+12 > len(t) {
			break
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 {
			o := int(bo.Uint16(t[e+8 : e+10]))
			if o >= 1 && o <= 8 {
				return o
			}
			return 1
		}
	}
	return 1
}

func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	ow, oh := w, h
	if o >= 5 && o <= 8 {
		ow, oh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.At(b.Min.X+x, b.Min.Y+y)
			switch o {
			case 2:
				dst.Set(w-1-x, y, c)
			case 3:
				dst.Set(w-1-x, h-1-y, c)
			case 4:
				dst.Set(x, h-1-y, c)
			case 5:
				dst.Set(y, x, c)
			case 6:
				dst.Set(h-1-y, x, c)
			case 7:
				dst.Set(oh-1-y, ow-1-x, c)
			case 8:
				dst.Set(y, w-1-x, c)
			}
		}
	}
	return dst
}
