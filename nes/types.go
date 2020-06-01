package gones

import (
	"github.com/tiagolobocastro/gones/nes/speakers"
	"image/color"
)

const (
	frameXWidth  = 256
	frameYHeight = 240

	screenFrameRatio = 3
	screenXWidth     = frameXWidth * screenFrameRatio
	screenYHeight    = frameYHeight * screenFrameRatio
)

type register struct {
	val uint8

	name string

	onWrite func()
	onRead  func() uint8
}

type register16 struct {
	val uint16

	name string
}

type ps_register struct {
	bit [8]byte

	name string
}

type spc_registers struct {
	pc register16
	sp register
	ps ps_register

	name string
}

type ix_registers struct {
	x register
	y register

	name string
}

type gp_registers struct {
	ac register
	ix ix_registers

	name string
}

type registers struct {
	spc     spc_registers
	gp      gp_registers
	verbose bool
}

const (
	// allows for validity test
	ModeInvalid = iota
	ModeZeroPage
	ModeIndexedZeroPageX
	ModeIndexedZeroPageY
	ModeAbsolute
	ModeIndexedAbsoluteX
	ModeIndexedAbsoluteY
	ModeIndirect
	ModeImplied
	ModeAccumulator
	ModeImmediate
	ModeRelative
	ModeIndexedIndirectX
	ModeIndirectIndexedY
)

type framebuffer struct {
	buffer0 []color.RGBA
	buffer1 []color.RGBA

	// 0 - backBuffer, 1 - frontBuffer
	frameIndex   int
	frameUpdated chan bool

	// number of frames
	frames int
}

type nes struct {
	bus

	cpu  Cpu
	ram  ram
	cart Cartridge
	ppu  Ppu
	dma  dma
	apu  Apu
	ctrl controllers

	screen screen

	resetRq bool

	// Options
	verbose  bool
	cartPath string
	freeRun  bool
	audioLib AudioLib
	audioLog bool
}

const (
	MapCPUId = iota
	MapPPUId
	MapDMAId
	MapAPUId
)

type iInterrupt interface {
	raise(uint8)
	clear(uint8)
}

type AudioLib string

const (
	Nil       = "nil"
	Beep      = "beep"
	PortAudio = "portaudio"
)

type AudioSpeaker interface {
	Init()
	Reset()
	Stop()
	Play()
	Sample(float64) bool
	SampleRate() int
	BufferReady() bool
}

func NewSpeaker(lib AudioLib) AudioSpeaker {
	var speaker AudioSpeaker
	switch lib {
	case Nil:
		speaker = new(speakers.SpeakerNil)
	case Beep:
		speaker = new(speakers.SpeakerBeep)
	case PortAudio:
		speaker = new(speakers.SpeakerPort)
	default:
		panic("Unknown speaker type!")
	}
	speaker.Init()
	return speaker
}

const NesBaseFrequency = 1789773
const NesApuFrequency = NesBaseFrequency / 2
const NesApuFrameCycles = 7457
const NesApuVolumeGain = 0.012

//const NesApuVolumeGain = 0.00752
