package main

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
)

func main() {
	dir := os.Args[1]
	n := 24
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &n)
	}
	_ = os.MkdirAll(dir, 0755)
	for i := 1; i <= n; i++ {
		w, h := 1600, 1200
		if i%3 == 0 {
			w, h = 1200, 1600
		}
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, color.RGBA{
					R: uint8((x + i*17) % 256),
					G: uint8((y + i*31) % 256),
					B: uint8((x/3 + y/5 + i*11) % 256),
					A: 255,
				})
			}
		}
		path := filepath.Join(dir, fmt.Sprintf("photo%02d.jpg", i))
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 88}); err != nil {
			panic(err)
		}
		f.Close()
	}
	fmt.Println("generated", n, "images in", dir)
}
