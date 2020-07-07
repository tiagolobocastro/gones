package ui

import (
	"fmt"
	"image/color"
	"os"
	"runtime"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"

	"github.com/tiagolobocastro/gones/lib/common"
)

const (
	screenFrameRatio = 3
	screenXWidth     = common.FrameXWidth * screenFrameRatio
	screenYHeight    = common.FrameYHeight * screenFrameRatio
)

type GoNes interface {
	Poke(controllerId uint8, button uint8, pressed bool)
	Request(request common.NesOpRequest)
}

type Screen struct {
	nes GoNes

	// window where we draw the sprite
	window *pixelgl.Window

	// front and back buffers
	buffer0 *pixel.PictureData
	buffer1 *pixel.PictureData
	sprite  *pixel.Sprite

	Framebuffer common.Framebuffer

	// FPS stats
	fpsChannel   <-chan time.Time
	fpsLastFrame int
}

func (s *Screen) Serialise(sr common.Serialiser) error {
	return sr.Serialise(s.buffer0, s.buffer1, s.Framebuffer.FrameIndex)
}
func (s *Screen) DeSerialise(sr common.Serialiser) error {
	return sr.DeSerialise(&s.buffer0, &s.buffer1, &s.Framebuffer.FrameIndex)
}

func (s *Screen) Init(nes GoNes) {
	s.nes = nes

	s.setSprite()
}

func (s *Screen) Run() {
	go func() {
		runtime.LockOSThread()
		pixelgl.Run(s.runThread)
		os.Exit(0)
	}()
}

func (s *Screen) runThread() {
	cfg := pixelgl.WindowConfig{
		Title:  "GoNes",
		Bounds: pixel.R(0, 0, screenXWidth, screenYHeight),
		VSync:  true,
	}
	window, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	s.window = window
	s.fpsChannel = time.Tick(time.Second)
	s.fpsLastFrame = 0

	s.runner()
}

func (s *Screen) runner() {
	lastLoopFrames := 0
	for !s.window.Closed() {

		<-s.Framebuffer.FrameUpdated

		frameDiff := s.Framebuffer.Frames - lastLoopFrames
		if frameDiff > 0 {
			if frameDiff > 1 {
				fmt.Printf("Oops, skipped %v frames!\n", frameDiff)
			}

			s.draw()
			s.window.Update()
			lastLoopFrames = s.Framebuffer.Frames
		}

		s.updateFpsTitle()
		s.updateControllers()
	}
	// wait for nes to halt on "sync"
	for len(s.Framebuffer.FrameUpdated) != 0 {
	}
	s.nes.Request(common.StopRequest)
}

var buttons = [8]struct {
	id  uint8
	key pixelgl.Button
}{
	{common.BitA, pixelgl.KeyS},
	{common.BitB, pixelgl.KeyA},
	{common.BitSelect, pixelgl.KeyLeftShift},
	{common.BitStart, pixelgl.KeyEnter},
	{common.BitUp, pixelgl.KeyUp},
	{common.BitDown, pixelgl.KeyDown},
	{common.BitLeft, pixelgl.KeyLeft},
	{common.BitRight, pixelgl.KeyRight},
}

func (s *Screen) updateControllers() {
	onePressed := false
	for _, button := range buttons {
		pressed := s.window.Pressed(button.key)
		s.nes.Poke(0, button.id, pressed)
		if pressed {
			onePressed = true
		}
	}

	if s.window.Pressed(pixelgl.KeyLeftControl) && s.window.JustPressed(pixelgl.KeyR) {
		s.nes.Request(common.ResetRequest)
		onePressed = true
	}
	if s.window.JustPressed(pixelgl.KeyLeftControl) && s.window.Pressed(pixelgl.KeyS) ||
		s.window.JustPressed(pixelgl.KeyS) && s.window.Pressed(pixelgl.KeyLeftControl) {
		s.nes.Request(common.SaveRequest)
		onePressed = true
	}
	if s.window.JustPressed(pixelgl.KeyLeftControl) && s.window.Pressed(pixelgl.KeyL) ||
		s.window.JustPressed(pixelgl.KeyL) && s.window.Pressed(pixelgl.KeyLeftControl) {
		s.nes.Request(common.LoadRequest)
		onePressed = true
	}

	if onePressed {
		s.window.UpdateInput()
	}
}

func (s *Screen) updateFpsTitle() {
	select {
	case <-s.fpsChannel:
		frames := s.Framebuffer.Frames - s.fpsLastFrame
		s.fpsLastFrame = s.Framebuffer.Frames

		s.window.SetTitle(fmt.Sprintf("GoNes | FPS: %d", frames))
	default:
	}
}

func (s *Screen) draw() {
	// seems to be required, for reasons unknown
	s.updateSprite()

	s.sprite.Draw(s.window, pixel.IM.Moved(s.window.Bounds().Center()).ScaledXY(s.window.Bounds().Center(), pixel.V(3, 3)))
}

func (s *Screen) updateSprite() {
	if s.Framebuffer.FrameIndex == 1 {
		// ppu is drawing new pixels on buffer1, which means the stable data is in buffer0
		s.sprite = pixel.NewSprite(s.buffer0, pixel.R(0, 0, common.FrameXWidth, common.FrameYHeight))
	} else {
		s.sprite = pixel.NewSprite(s.buffer1, pixel.R(0, 0, common.FrameXWidth, common.FrameYHeight))
	}
}

func (s *Screen) setSprite() {

	s.buffer0 = &pixel.PictureData{
		Pix:    make([]color.RGBA, common.FrameXWidth*common.FrameYHeight),
		Stride: common.FrameXWidth,
		Rect:   pixel.R(0, 0, common.FrameXWidth, common.FrameYHeight),
	}

	s.buffer1 = &pixel.PictureData{
		Pix:    make([]color.RGBA, common.FrameXWidth*common.FrameYHeight),
		Stride: common.FrameXWidth,
		Rect:   pixel.R(0, 0, common.FrameXWidth, common.FrameYHeight),
	}

	s.Framebuffer = common.Framebuffer{
		Buffer0:      s.buffer0.Pix,
		Buffer1:      s.buffer1.Pix,
		FrameIndex:   0,
		FrameUpdated: make(chan bool),
		Frames:       0,
	}

	s.updateSprite()
}
