package common

import "image/color"

type IiInterrupt interface {
	Raise(uint8)
	Clear(uint8)
}

type Framebuffer struct {
	Buffer0 []color.RGBA
	Buffer1 []color.RGBA

	// 0 - backBuffer, 1 - frontBuffer
	FrameIndex   int
	FrameUpdated chan bool

	// number of frames
	Frames int
}
