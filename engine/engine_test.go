package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / w), G: uint8(y * 255 / h),
				B: uint8((x + y) * 255 / (w + h)), A: 255,
			})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestNaturalLessOrder(t *testing.T) {
	got := []string{"photo10.jpg", "photo2.jpg", "photo1.jpg", "Photo3.JPG"}
	for i := 1; i < len(got); i++ {
		for j := i; j > 0 && naturalLess(got[j], got[j-1]); j-- {
			got[j], got[j-1] = got[j-1], got[j]
		}
	}
	want := []string{"photo1.jpg", "photo2.jpg", "Photo3.JPG", "photo10.jpg"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("natural order: got %v, want %v", got, want)
		}
	}
}

func TestPadJPEGTail(t *testing.T) {
	raw := testJPEG(t, 32, 32)
	patched := padJPEGTail(raw, 64)
	if patched == nil || len(patched) != len(raw)+64 {
		t.Fatal("padJPEGTail failed")
	}
	if !bytes.HasSuffix(patched, []byte{0xFF, 0xD9}) {
		t.Fatal("EOI marker lost")
	}
}

func brokenHeightJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	raw := testJPEG(t, w, h)
	for i := 0; i+9 < len(raw); i++ {
		if raw[i] != 0xFF || (raw[i+1] != 0xC0 && raw[i+1] != 0xC2) {
			continue
		}
		old := binary.BigEndian.Uint16(raw[i+5 : i+7])
		binary.BigEndian.PutUint16(raw[i+5:i+7], old+8)
		return raw
	}
	t.Fatal("SOF marker not found")
	return nil
}

func TestDecodeImageRepairsTruncatedJPEG(t *testing.T) {
	raw := brokenHeightJPEG(t, 64, 64)
	img, repaired, err := decodeImage(raw)
	if err != nil || !repaired || img.Bounds().Dy() != 72 {
		t.Fatalf("repair failed: repaired=%v err=%v", repaired, err)
	}
}

func TestProcessImageRespectsMaxEdge(t *testing.T) {
	p, err := processImage(testJPEG(t, 400, 200), 200, 200*1024, 82)
	if err != nil {
		t.Fatal(err)
	}
	if p.w != 200 || p.h != 100 {
		t.Fatalf("scaled to %dx%d, want 200x100", p.w, p.h)
	}
	if len(p.jpeg) == 0 || len(p.jpeg) > 200*1024 {
		t.Fatalf("unexpected jpeg size %d", len(p.jpeg))
	}
}

func TestReadDPI(t *testing.T) {
	x, y := readDPI(testJPEG(t, 16, 16))
	if x != 96 || y != 96 {
		t.Fatalf("default DPI = %v/%v, want 96/96", x, y)
	}
}

func TestWriteErrorReportNeverWritesPasswords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out_error_report.log")
	o := Options{Source: "/private/photos.rar", ArchivePass: "ARCHIVE-SECRET",
		PDFPass: "PDF-SECRET", OutPath: "/private/out.pdf", TargetMB: 20,
		MaxEdge: 1600, Quality: 82, PageMode: "fit", Workers: 4, Lang: "en"}
	if err := writeErrorReport(path, o, []Msg{M("engine.skipped", "Image skipped")}, 1, fmt.Errorf("fatal")); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	for _, secret := range []string{o.ArchivePass, o.PDFPass} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("password leaked: %s", secret)
		}
	}
	removeStaleReport(path)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("stale report not removed")
	}
}

func TestRunJobEndToEndWithBookmarksAndPaper(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "images")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	for i, size := range []struct{ w, h int }{{320, 160}, {160, 320}, {200, 200}} {
		name := fmt.Sprintf("photo%d.jpg", i+1)
		if err := os.WriteFile(filepath.Join(src, name), testJPEG(t, size.w, size.h), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(src, "bad.jpg"), []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "result.pdf")
	o := Options{
		Source: src, PDFPass: "OpenSesame", OutPath: out,
		MaxEdge: 200, TargetMB: 5, Quality: 82,
		CompressionMode: "target-mb", PageMode: "paper", Paper: "A4",
		Bookmarks: true, Title: "测试合集", Workers: 2, Lang: "en",
	}
	j := newJob()
	path, size, pages, _, err := runJob(o, j)
	if err != nil || pages != 3 {
		t.Fatalf("runJob: pages=%d err=%v", pages, err)
	}
	if path != out || size <= 0 || size > int64(o.TargetMB)*1024*1024 {
		t.Fatalf("unexpected output %s (%d bytes)", path, size)
	}
	if len(j.getWarnings()) != 1 {
		t.Fatalf("expected exactly one skipped image, got %d", len(j.getWarnings()))
	}
	data, _ := os.ReadFile(out)
	for _, want := range []string{"%PDF-", "/Encrypt", "/DCTDecode", "/MediaBox", "/Outlines", "测试合集"} {
		if !bytes.Contains(data, []byte(want)) {
			t.Fatalf("PDF missing %q", want)
		}
	}
}
