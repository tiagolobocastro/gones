package gones

import (
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"image/color"
	_ "image/png" // ouch! needs to be here
	"log"
	"os"
	"runtime"
	"time"
)

type screen struct {
	window *pixelgl.Window
	sprite *pixel.Sprite

	nes *nes
	pix *pixel.PictureData

	freeRun bool

	fpsChannel   <-chan time.Time
	fpsLastFrame int
}

func (s *screen) init(nes *nes) {
	s.nes = nes

	s.setSprite()

	if nes.cart.cart == "" {
		return
	}
}

func (s *screen) run(freeRun bool) {
	go func() {
		s.freeRun = freeRun
		runtime.LockOSThread()
		pixelgl.Run(s.runThread)
		os.Exit(0)
	}()
}

func (s *screen) runThread() {

	cfg := pixelgl.WindowConfig{
		Title:  "GoNes",
		Bounds: pixel.R(0, 0, 510, 480),
		VSync:  true,
	}
	window, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	s.window = window
	s.fpsChannel = time.Tick(time.Second)
	s.fpsLastFrame = 0

	if s.freeRun {
		s.freeRunner()
	} else {
		s.runner()
	}
}

func (s *screen) runner() {
	lastLoopStamp := time.Now()
	lastLoopFrames := 0

	for !s.window.Closed() {

		dt := time.Since(lastLoopStamp).Seconds()
		lastLoopStamp = time.Now()

		// not quite right... if we click the window, it cause issues -> leads us to execute
		// in big "chunks" and therefore loosing frames
		// also same in debug mode...
		// 0.02 seems to be small enough to make this imperceptible and allowing it to "catch up"
		// doesn't work for debug as the nes step is slower, increasing the dt
		if dt < 0.02 {
			s.nes.Step(dt)
		}

		// not good at the moment because the window updates do not match the ppu steps, unless we make sure
		// we always break out of the step after a vblank?
		// perhaps would be better if we run the nes on a separate thread and use channels to control when the
		// nes can execute?

		// draw only after the ppu is finished poking the pixels -> after vblank when we increment the frames
		// todo: use the interrupt interconnect to detect this
		if s.nes.ppu.frames > lastLoopFrames {
			s.draw()
			s.window.Update()
			lastLoopFrames = s.nes.ppu.frames
		}

		s.updateFpsTitle()
		s.updateControllers()
	}
}

func (s *screen) updateControllers() {

	for _, button := range [8]struct {
		id  uint8
		key pixelgl.Button
	}{
		{bitA, pixelgl.KeyS},
		{bitB, pixelgl.KeyD},
		{bitSelect, pixelgl.KeyC},
		{bitStart, pixelgl.KeySpace},
		{bitUp, pixelgl.KeyUp},
		{bitDown, pixelgl.KeyDown},
		{bitLeft, pixelgl.KeyLeft},
		{bitRight, pixelgl.KeyRight},
	} {
		pressed := s.window.Pressed(button.key)
		s.nes.ctrl.poke(0, button.id, pressed)
	}
}

func (s *screen) updateFpsTitle() {
	select {
	case <-s.fpsChannel:
		frames := s.nes.ppu.frames - s.fpsLastFrame
		s.fpsLastFrame = s.nes.ppu.frames

		s.window.SetTitle(fmt.Sprintf("%s | FPS: %d", "GoNes", frames))
	default:
	}
}

func (s *screen) freeRunner() {
	lastLoopFrames := 0
	for !s.window.Closed() {
		// draw only after the ppu is finished poking the pixels -> after vblank when we increment the frames
		// todo: use the interrupt interconnect to detect this
		if s.nes.ppu.frames > lastLoopFrames {
			s.draw()
			s.window.Update()
			lastLoopFrames = s.nes.ppu.frames
		}

		s.updateFpsTitle()
	}
}

func (s *screen) draw() {
	// seems to be required for reasons unknown
	s.updateSprite()
	s.sprite.Draw(s.window, pixel.IM.Moved(s.window.Bounds().Center()).ScaledXY(s.window.Bounds().Center(), pixel.V(2, 2)))
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
