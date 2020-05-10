package gones

import (
	"fmt"
	"log"
	"time"

	"github.com/tiagolobocastro/gones/nes/waves"
)

const NesApuFrequency = NesBaseFrequency / 2
const NesApuFrameCycles = 7457
const NesApuVolumeGain = 0.012

type Apu struct {
	pulse1 waves.Pulse
	pulse2 waves.Pulse

	clock   uint
	verbose bool

	frameCounter uint
	frameStep    uint
	frameMode    uint

	speaker   SpeakerBeep
	startTime time.Time
	samples   uint64
}

func (a *Apu) reset() {
	a.init(nil, a.verbose)
}
func (a *Apu) init(busInt busExtInt, verbose bool) {
	a.verbose = verbose

	a.pulse1.Init(true)
	a.pulse2.Init(false)

	a.speaker.init()

	a.startTime = time.Now()
	a.samples = 0

	a.clock = 0
	a.frameCounter = 0
	a.frameStep = 0
	a.frameMode = 0
}

func (a *Apu) SamplingRate() float64 {
	return float64(NesBaseFrequency) / float64(a.speaker.SampleRate())
}
func (a *Apu) addSample(val float64) {
	select {
	case a.speaker.sampleChan <- val:
	default:
		secs := time.Since(a.startTime).Seconds()
		sps := a.samples / uint64(secs)
		fmt.Printf("%d and %v: %d Hz\n", a.samples, secs, sps)
	}

	a.samples++
}

func (a *Apu) ticks(nTicks int) {
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
		// todo: change pulse to use CPU cycles instead!
		a.pulse1.Tick()
		a.pulse2.Tick()

		a.sample()
	}
}

func (a *Apu) sample() {
	if (a.clock % uint(a.SamplingRate())) == 0 {
		mix := a.mixPulses(a.pulse1.Sample(), a.pulse2.Sample())
		a.addSample(mix)
	}
}

func (a *Apu) triangleLinearTick() {
}
func (a *Apu) quarterFrameTick() {
	a.triangleLinearTick()
}

func (a *Apu) halfFrameTick() {
	a.pulse1.HalfFrameTick()
	a.pulse2.HalfFrameTick()
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

func (a *Apu) read8(addr uint16) uint8 {
	log.Printf("Error: Reading from the APU addr %x\n", addr)
	return 0
}
func (a *Apu) write8(addr uint16, val uint8) {
	switch {
	case addr >= 0x4000 && addr <= 0x4003:
		a.pulse1.Write8(addr, val)
	case addr >= 0x4004 && addr <= 0x4007:
		a.pulse2.Write8(addr, val)
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
	//pulseOut := 0.00752 * (pulse1 + pulse2)
	pulseOut := NesApuVolumeGain * (pulse1 + pulse2)
	return pulseOut
}
