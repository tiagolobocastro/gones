package apu

import (
	"fmt"
	"log"
	"time"

	"github.com/tiagolobocastro/gones/lib/apu/waves"
	"github.com/tiagolobocastro/gones/lib/common"
	"github.com/tiagolobocastro/gones/lib/cpu"
	"github.com/tiagolobocastro/gones/lib/speakers"
)

// this should be passed through via Init
const NesBaseFrequency = 1789773 // NTSC
const NesApuFrameCycles = 7457
const NesApuVolumeGain = 0.012

// Status Registers Enable bits
// ---D NT21 Enable DMC (D), noise (N), triangle (T), and pulse channels (2/1)
const (
	bP1 = 1 << 0
	bP2 = 1 << 1
	bT  = 1 << 2
	bN  = 1 << 3
	bD  = 1 << 4
)

func (a *Apu) writeStatusReg() {
	a.pulse1.Enable((a.status.Val & bP1) != 0)
	a.pulse2.Enable((a.status.Val & bP2) != 0)
	a.triangle.Enable((a.status.Val & bT) != 0)
	a.noise.Enable((a.status.Val & bN) != 0)
	a.dmc.Enable((a.status.Val & bD) != 0)
	// Clear the DMC interrupt flag
}
func (a *Apu) readStatusReg() uint8 {
	status := uint8(0)
	if a.pulse1.Enabled() {
		status |= bP1
	}
	if a.pulse2.Enabled() {
		status |= bP2
	}
	if a.triangle.Enabled() {
		status |= bT
	}
	if a.noise.Enabled() {
		status |= bN
	}
	if a.dmc.Enabled() {
		status |= bD
	}
	// Reading this register clears the frame interrupt flag
	// (but not the DMC interrupt flag).
	// If an interrupt flag was set at the same moment of the read, it will
	// read back as 1 but it will not be cleared.
	return status
}

type Apu struct {
	common.BusInt
	interrupts common.IiInterrupt

	pulse1 waves.Pulse
	pulse2 waves.Pulse

	triangle waves.Triangle

	noise waves.Noise

	dmc waves.Dmc

	clock   uint
	verbose bool
	enabled bool

	frameCounter uint
	frameStep    uint
	frameMode    uint
	frameIrqEn   bool

	logAudio      bool
	samples       uint
	sampleLogTime time.Time
	samplesTotal  uint

	status common.Register

	audioLib speakers.AudioLib
	speaker  speakers.AudioSpeaker

	sampleTicks       float64
	sampleTargetTicks float64
}

func (a *Apu) Serialise(s common.Serialiser) error {
	return s.Serialise(
		&a.pulse1, &a.pulse2, &a.triangle, &a.noise, &a.dmc,
		a.clock, a.enabled, a.frameCounter, a.frameStep, a.frameMode, a.frameIrqEn,
		a.status, a.sampleTicks, a.sampleTargetTicks, a.samples, a.samplesTotal,
		a.sampleLogTime,
	)
}
func (a *Apu) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&a.pulse1, &a.pulse2, &a.triangle, &a.noise, &a.dmc,
		&a.clock, &a.enabled, &a.frameCounter, &a.frameStep, &a.frameMode, &a.frameIrqEn,
		&a.status, &a.sampleTicks, &a.sampleTargetTicks, &a.samples, &a.samplesTotal,
		&a.sampleLogTime,
	)
}

func (a *Apu) Reset() {
	if !a.enabled {
		return
	}

	a.pulse1.Init(true)
	a.pulse2.Init(false)
	a.triangle.Init()
	a.noise.Init()
	a.dmc.Init(a.BusInt)

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
	a.frameIrqEn = true

	a.status.Initx("status", 0, a.writeStatusReg, a.readStatusReg)
}
func (a *Apu) Init(busInt common.BusInt, interrupts common.IiInterrupt, verbose bool, logAudio bool, audioLib speakers.AudioLib) {
	a.BusInt = busInt
	a.interrupts = interrupts

	a.verbose = verbose
	a.logAudio = logAudio
	a.audioLib = audioLib
	a.enabled = true
	a.speaker = speakers.NewSpeaker(a.audioLib)

	a.Reset()
}
func (a *Apu) Play() {
	a.speaker.Play()
}
func (a *Apu) Stop() {
	a.Reset()
	a.enabled = false
	a.speaker.Stop()
}

var lastLagReported time.Time

func (a *Apu) addSample(val float64) {
	if !a.speaker.Sample(val) {
		if time.Now().Second()-lastLagReported.Second() > 1 {
			lastLagReported = time.Now()
			go fmt.Printf("The Audio Speaker is falling behind the audio samples!\n")
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
		go fmt.Printf("Sampling: Real %v Hz, Apu %v Hz\n", sps, hz)
	}
}

func (a *Apu) Ticks(nTicks int) {
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
		a.dmc.Tick()
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
		dmc := a.dmc.Sample()
		//dmc := 0.0
		mix := 0.00851*triangle + 0.00494*noise + 0.00335*dmc + mixPulses

		a.addSample(mix)
	}
}

func (a *Apu) AudioBufferReady() bool {
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

// mode 0:    mode 1:       function
// ---------  -----------  -----------------------------
//  - - - f    - - - - -    IRQ (if bit 6 is clear)
//  - l - l    - l - - l    Length counter and sweep
//  e e e e    e e e - e    Envelope and linear counter
func (a *Apu) frameTick() {
	a.frameCounter++

	if a.frameCounter == NesApuFrameCycles {
		a.frameCounter = 0

		if a.frameMode == 0 {
			// 4 step sequence

			// clock envelopes and triangle linear counter
			a.quarterFrameTick()
			if a.frameStep == 0 {
				// set int flag if inhibit is clear
				a.raiseIrq()
			}
		} else {
			// 5 step sequence
			a.frameStep = (a.frameStep + 1) % 5
			if a.frameStep != 3 {
				// clock envelopes and triangle linear counter
				a.quarterFrameTick()
			}
		}

		if a.frameStep == 1 || a.frameStep == (a.frameMode+3) {
			// clock len counters and sweep units
			a.halfFrameTick()
		}

		a.frameStep = (a.frameStep + 1) % (a.frameMode + 4)
	}
}
func (a *Apu) raiseIrq() {
	if a.frameIrqEn {
		a.interrupts.Raise(cpu.CpuIntIRQ)
	}
}

func (a *Apu) Read8(addr uint16) uint8 {
	switch {
	case addr == 0x4015:
		return a.status.Read()
	default:
		log.Printf("Error: Reading from the APU addr %X\n", addr)
	}
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
	case addr >= 0x4010 && addr <= 0x4013:
		a.dmc.Write8(addr, val)
	case addr == 0x4015:
		a.status.Write(val)
	case addr == 0x4017:
		a.frameMode = uint(val & 0x80)
		a.frameIrqEn = (val & 0x40) == 0
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
