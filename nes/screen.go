package gones

import (
	"fmt"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"image/color"
	"os"
	"runtime"
	"time"
)

type screen struct {
	nes *nes

	// window where we draw the sprite
	window *pixelgl.Window

	// front and back buffers
	buffer0 *pixel.PictureData
	buffer1 *pixel.PictureData
	sprite  *pixel.Sprite

	framebuffer framebuffer

	// free run -> no vsync
	freeRun bool

	// FPS stats
	fpsChannel   <-chan time.Time
	fpsLastFrame int
}

func (s *screen) init(nes *nes) {
	s.nes = nes

	s.setSprite()
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

	go func() {
		tmr := time.Tick(time.Second / 240)
		for {
			s.nes.Step((time.Second / 240).Seconds())
			<-tmr
		}
	}()

	lastLoopFrames := 0
	for !s.window.Closed() {

		<-s.framebuffer.frameUpdated

		frameDiff := s.nes.ppu.frames - lastLoopFrames
		if frameDiff > 0 {
			if frameDiff > 1 {
				fmt.Printf("Ups, skipped %v frames!\n", frameDiff)
			}

			s.draw()
			s.window.Update()
			lastLoopFrames = s.nes.ppu.frames
		}

		s.updateFpsTitle()
		s.updateControllers()
	}
}

var buttons = [8]struct {
	id  uint8
	key pixelgl.Button
}{
	{bitA, pixelgl.KeyD},
	{bitB, pixelgl.KeyF},
	{bitSelect, pixelgl.KeyS},
	{bitStart, pixelgl.KeyEnter},
	{bitUp, pixelgl.KeyUp},
	{bitDown, pixelgl.KeyDown},
	{bitLeft, pixelgl.KeyLeft},
	{bitRight, pixelgl.KeyRight},
}

func (s *screen) updateControllers() {

	for _, button := range buttons {
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
	for !s.window.Closed() {
		s.updateFpsTitle()
		s.updateControllers()
	}
}

func (s *screen) draw() {
	// seems to be required, for reasons unknown
	s.updateSprite()

	s.sprite.Draw(s.window, pixel.IM.Moved(s.window.Bounds().Center()).ScaledXY(s.window.Bounds().Center(), pixel.V(2, 2)))
}

func (s *screen) updateSprite() {
	if s.framebuffer.frameIndex == 1 {
		// ppu is drawing new pixels on buffer1, which means the stable data is in buffer0
		s.sprite = pixel.NewSprite(s.buffer0, pixel.R(0, 0, screenXWidth, screenYWidth))
	} else {
		s.sprite = pixel.NewSprite(s.buffer1, pixel.R(0, 0, screenXWidth, screenYWidth))
	}
}

func (s *screen) setSprite() {

	s.buffer0 = &pixel.PictureData{
		Pix:    make([]color.RGBA, screenXWidth*screenYWidth),
		Stride: screenXWidth,
		Rect:   pixel.R(0, 0, screenXWidth, screenYWidth),
	}

	s.buffer1 = &pixel.PictureData{
		Pix:    make([]color.RGBA, screenXWidth*screenYWidth),
		Stride: screenXWidth,
		Rect:   pixel.R(0, 0, screenXWidth, screenYWidth),
	}

	s.framebuffer = framebuffer{
		buffer0:      s.buffer0.Pix,
		buffer1:      s.buffer1.Pix,
		frameIndex:   0,
		frameUpdated: make(chan bool),
		frames:       0,
	}

	s.updateSprite()
}
