package gones

import (
	"fmt"
	"image/color"
	_ "image/png" // ouch!
	"log"
	"os"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

func run() {
	cfg := pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	pic := &pixel.PictureData{
		Pix:    make([]color.RGBA, 32*32),
		Stride: 8,
		Rect:   pixel.R(0, 0, 32, 32),
	}

	file, err := os.Open("mario.chr") // For read access.
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, 10000)
	count, err := file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("read %d bytes: %q\n", count, data[:count])

	palette := [4]color.RGBA{
		{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, // B - W
		{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}, // 1 - R
		{R: 0x00, G: 0x00, B: 0xFF, A: 0xFF}, // 2 - B
		{R: 0x00, G: 0xF0, B: 0xF0, A: 0xFF}, // 3 - LB
	}

	for y := uint(0); y < 8; y++ {
		for x := uint(0); x < 8; x++ {

			i := (data[y] >> (8 - 1 - x)) & 1
			j := (data[y+8] >> (8 - 1 - x)) & 1
			rgb := palette[j<<1|i]
			pic.Pix[(8-1-y)*8+x] = rgb
		}
	}

	spr := pixel.NewSprite(pic, pixel.R(0, 0, 8, 8))

	for !win.Closed() {
		win.Clear(colornames.Whitesmoke)
		spr.Draw(win, pixel.IM.Moved(win.Bounds().Center()).ScaledXY(win.Bounds().Center(), pixel.V(8, 8)))

		win.Update()
	}
}

func Start() {
	pixelgl.Run(run)
}
