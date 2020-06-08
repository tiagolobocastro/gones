package gones

import (
	"github.com/tiagolobocastro/gones/nes/common"
	cpu2 "github.com/tiagolobocastro/gones/nes/cpu"
	"github.com/tiagolobocastro/gones/nes/mappers"
	ppu2 "github.com/tiagolobocastro/gones/nes/ppu"
	"github.com/tiagolobocastro/gones/nes/speakers"
)

const (
	frameXWidth  = 256
	frameYHeight = 240

	screenFrameRatio = 3
	screenXWidth     = frameXWidth * screenFrameRatio
	screenYHeight    = frameYHeight * screenFrameRatio
)

type nes struct {
	bus common.Bus

	cpu  cpu2.Cpu
	ram  common.Ram
	cart mappers.Cartridge
	ppu  ppu2.Ppu
	dma  dma
	apu  Apu
	ctrl controllers

	screen screen

	resetRq bool

	// Options
	verbose     bool
	cartPath    string
	freeRun     bool
	audioLib    AudioLib
	audioLog    bool
	spriteLimit bool
}

const (
	MapCPUId = iota
	MapPPUId
	MapDMAId
	MapAPUId
)

type AudioLib string

const (
	Nil       = "nil"
	Beep      = "beep"
	PortAudio = "portaudio"
	Oto       = "oto"
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
	case Oto:
		speaker = new(speakers.SpeakerOto)
	default:
		panic("Unknown speaker type!")
	}
	speaker.Init()
	return speaker
}

const NesBaseFrequency = 1789773 // NTSC
//const NesBaseFrequency = 1662607 // PAL

//const NesApuFrequency = NesBaseFrequency / 2
const NesApuFrameCycles = 7457
const NesApuVolumeGain = 0.012

//const NesApuVolumeGain = 0.00752
