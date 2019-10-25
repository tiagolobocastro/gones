package gones

import (
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
	"image/color"
	_ "image/png" // ouch! needs to be here
	"log"
	"os"
	"runtime"
)

type screen struct {
	window *pixelgl.Window
	sprite *pixel.Sprite

	nes         *nes
	pix         *pixel.PictureData
	frameBuffer *[]color.RGBA
}

func (s *screen) init(nes *nes) {
	s.nes = nes

	s.setSprite()

	if nes.cart.cart == "" {
		return
	}

	go func() {
		runtime.LockOSThread()
		pixelgl.Run(s.run)
		os.Exit(0)
	}()
}

func (s *screen) run() {

	cfg := pixelgl.WindowConfig{
		Title:  "GoNes",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
	window, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	s.window = window

	s.runner()
}

func (s *screen) runner() {

	for !s.window.Closed() {
		win := s.window

		e := (s.nes.ppu.regs[PPUMASK].val & 0xE0) >> 5

		emphasis_table := []color.Color{
			colornames.Whitesmoke,
			/* 001 Red      */ colornames.Red, // 1239,  915,  743,
			/* 010 Green    */ colornames.Green, //  794, 1086,  882,
			/* 011 Yellow   */ colornames.Yellow, // 1019,  980,  653,
			/* 100 Blue     */ colornames.Blue, //  905, 1026, 1277,
			/* 101 Magenta  */ colornames.Magenta, // 1023,  908,  979,
			/* 110 Cyan     */ colornames.Cyan, //  741,  987, 1001,
			/* 111 Black    */ colornames.Black, //  750,  750,  750
		}

		// this is wrong btw, but it's a nice way to test the nes atm
		// todo: how to apply the emphasis!?
		win.Clear(emphasis_table[e])

		s.sprite.Draw(win, pixel.IM.Moved(win.Bounds().Center()).ScaledXY(win.Bounds().Center(), pixel.V(2, 2)))
		win.Update()

		//time.Sleep(time.Millisecond * 1)
		s.updateSprite()
	}
}

func (s *screen) updateSprite() {
	s.sprite = pixel.NewSprite(s.pix, pixel.R(0, 0, 256, 240))
}

func (s *screen) setSprite() {

	s.pix = &pixel.PictureData{
		Pix:    make([]color.RGBA, 256*240),
		Stride: 256,
		Rect:   pixel.R(0, 0, 256, 240),
	}

	s.sprite = pixel.NewSprite(s.pix, pixel.R(0, 0, 256, 240))
}

func (s *screen) addSpriteX(X uint) {

	pic := &pixel.PictureData{
		Pix:    make([]color.RGBA, 8*8),
		Stride: 8,
		Rect:   pixel.R(0, 0, 8, 8),
	}

	file, err := os.Open("mario.chr") // For read access.
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, 10000)
	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}

	palette := [4]color.RGBA{
		{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, // B - W
		{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}, // 1 - R
		{R: 0x00, G: 0x00, B: 0xFF, A: 0xFF}, // 2 - B
		{R: 0x00, G: 0xF0, B: 0xF0, A: 0xFF}, // 3 - LB
	}

	for y := uint(0); y < 8; y++ {
		for x := uint(0); x < 8; x++ {

			i := (data[y+X] >> (8 - 1 - x - X)) & 1
			j := (data[y+8+X] >> (8 - 1 - x - X)) & 1
			rgb := palette[j<<1|i]
			pic.Pix[(8-1-y)*8+x] = rgb
		}
	}

	s.sprite = pixel.NewSprite(pic, pixel.R(0, 0, 8, 8))
}
