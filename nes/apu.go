package gones

import (
	"fmt"
	"log"
	"time"

	"github.com/tiagolobocastro/gones/nes/waves"
)

type Apu struct {
	pulse1 waves.Pulse
	pulse2 waves.Pulse

	triangle waves.Triangle

	noise waves.Noise

	clock   uint
	verbose bool
	enabled bool

	frameCounter uint
	frameStep    uint
	frameMode    uint

	logAudio      bool
	samples       uint
	sampleLogTime time.Time
	samplesTotal  uint

	audioLib AudioLib
	speaker  AudioSpeaker

	sampleTicks       float64
	sampleTargetTicks float64
}

func (a *Apu) reset() {
	if !a.enabled {
		return
	}

	a.pulse1.Init(true)
	a.pulse2.Init(false)
	a.triangle.Init()
	a.noise.Init()

	a.speaker.Reset()
	a.sampleTicks = float64(NesBaseFrequency) / float64(a.speaker.SampleRate())
	a.sampleTargetTicks = a.sampleTicks

	a.sampleLogTime = time.Now()
	a.samples = 0
	a.samplesTotal = 0

	a.clock = 0
	a.frameCounter = 0
	a.frameStep = 0
	a.frameMode = 0
}
func (a *Apu) init(verbose bool, logAudio bool, audioLib AudioLib) {
	a.verbose = verbose
	a.logAudio = logAudio
	a.audioLib = audioLib
	a.enabled = true
	a.speaker = NewSpeaker(a.audioLib)

	a.reset()
}
func (a *Apu) Play() {
	a.speaker.Play()
}
func (a *Apu) Stop() {
	a.reset()
	a.enabled = false
	a.speaker.Stop()
}

var lastLagReported time.Time

func (a *Apu) addSample(val float64) {
	if !a.speaker.Sample(val) {
		if time.Now().Second()-lastLagReported.Second() > 1 {
			lastLagReported = time.Now()
			fmt.Printf("The Audio Speaker is falling behind the audio samples!\n")
		}
	}
	a.logSampling()
}
func (a *Apu) logSampling() {
	a.samples++
	a.samplesTotal++

	if !a.logAudio {
		return
	}

	if (a.samples % uint(a.speaker.SampleRate())) == 0 {
		sps := float64(a.samples) / time.Since(a.sampleLogTime).Seconds()
		a.sampleLogTime = time.Now()
		hz := NesBaseFrequency / (float64(a.clock) / float64(a.samplesTotal))
		a.samples = 0
		fmt.Printf("Sampling: Real %v Hz, Apu %v Hz\n", sps, hz)
	}
}

func (a *Apu) ticks(nTicks int) {
	if !a.enabled {
		return
	}

	for i := 0; i < nTicks; i++ {
		a.tick()
	}
}
func (a *Apu) tick() {
	a.clock++

	// APU is clocked every other CPU cycle
	// the frame counter is clocked every 3728.5 clocks
	// in other words, every 7457 CPU clocks
	// so emulate the APU using CPU clock cycles with the
	// necessary modification
	a.frameTick()
	if (a.clock % 2) == 0 {
		a.pulse1.Tick()
		a.pulse2.Tick()
		a.noise.Tick()
	}
	a.triangle.Tick()
	a.sample()
}

func (a *Apu) sample() {
	if a.clock >= uint(a.sampleTargetTicks) {
		a.sampleTargetTicks += a.sampleTicks

		mixPulses := a.mixPulses(a.pulse1.Sample(), a.pulse2.Sample())
		//mixPulses := 0.0
		triangle := a.triangle.Sample()
		//triangle := 0.0
		noise := a.noise.Sample()
		//noise := 0.0
		mix := 0.00851*triangle + 0.00494*noise + mixPulses

		a.addSample(mix)
	}
}

func (a *Apu) audioBufferReady() bool {
	return a.speaker.BufferReady()
}

func (a *Apu) quarterFrameTick() {
	a.pulse1.QuarterFrameTick()
	a.pulse2.QuarterFrameTick()
	a.triangle.QuarterFrameTick()
	a.noise.QuarterFrameTick()
}

func (a *Apu) halfFrameTick() {
	a.pulse1.HalfFrameTick()
	a.pulse2.HalfFrameTick()
	a.triangle.HalfFrameTick()
	a.noise.HalfFrameTick()
}

func (a *Apu) frameTick() {
	a.frameCounter++

	if a.frameCounter == NesApuFrameCycles {
		a.frameCounter = 0

		if a.frameMode == 0 {
			// 4 step sequence
			a.frameStep = (a.frameStep + 1) % 4
			// clock envelopes and triangle linear counter
			a.quarterFrameTick()
			if a.frameStep == 0 {
				// set int flag if inhibit is clear
			}
		} else {
			// 5 step sequence
			a.frameStep = (a.frameStep + 1) % 5
			// clock envelopes and triangle linear counter
			if a.frameStep != 4 {
				a.quarterFrameTick()
			}
		}

		if a.frameStep == 2 || a.frameStep == 0 {
			// clock len counters and sweep units
			a.halfFrameTick()
		}
	}
}

func (a *Apu) Read8(addr uint16) uint8 {
	log.Printf("Error: Reading from the APU addr %X\n", addr)
	return 0
}
func (a *Apu) Write8(addr uint16, val uint8) {
	switch {
	case addr >= 0x4000 && addr <= 0x4003:
		a.pulse1.Write8(addr, val)
	case addr >= 0x4004 && addr <= 0x4007:
		a.pulse2.Write8(addr, val)
	case addr == 0x4008, addr == 0x4009:
		a.triangle.Write8(addr, val)
	case addr == 0x400A, addr == 0x400B:
		a.triangle.Write8(addr, val)
	case addr >= 0x400C && addr <= 0x400F:
		a.noise.Write8(addr, val)
	case addr == 0x4017:
		a.frameMode = uint(val & 0x80)
		a.frameStep = 0
		a.frameCounter = 0
		if a.frameMode != 0 {
			a.quarterFrameTick()
			a.halfFrameTick()
		}
	}
}

func (a *Apu) mixPulses(pulse1 float64, pulse2 float64) float64 {
	//pulseOut := 95.88 / ((8128 / (pulse1 + pulse2)) + 100)
	pulseOut := NesApuVolumeGain * (pulse1 + pulse2)
	return pulseOut
}
