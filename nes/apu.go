package gones

import (
	"fmt"
	"log"
	"time"
)

const (
	channelPulse1 = iota
	channelPulse2
	channelTriangle
	channelNoise
	channelDMC
	channelAll1
	channelAll2
)

const nesApuFrequency = nesBaseFrequency / 2
const nesApuVolumeGain = 0.015

type Apu struct {
	pulse1 Pulse
	pulse2 Pulse

	clock   uint
	verbose bool

	speaker   SpeakerBeep
	startTime time.Time
	samples   uint64
}

func (a *Apu) AddSample(val float64) {
	select {
	case a.speaker.sampleChan <- val:
	default:
		secs := time.Since(a.startTime).Seconds()
		sps := a.samples / uint64(secs)
		fmt.Printf("%d and %v: %d Hz\n", a.samples, secs, sps)
	}

	a.samples++
}

func (a *Apu) reset() {
	a.init(nil, a.verbose)
}
func (a *Apu) init(busInt busExtInt, verbose bool) {
	a.verbose = verbose

	a.pulse1.init(true, a)
	a.pulse2.init(false, a)

	a.speaker.init()

	a.startTime = time.Now()
	a.samples = 0
}

func (a *Apu) ticks(nTicks int) {
	for i := 0; i < nTicks; i++ {
		a.tick()
	}
}
func (a *Apu) tick() {
	a.clock++

	// apu is clocked every other cpu clock
	if (a.clock % 2) == 0 {
		a.pulse1.tick()
	}
}

type Sequencer struct {
	clock uint

	timer  uint16 // 11bit timer
	reload uint16
}

func (s *Sequencer) set(timer uint16) {
	s.timer = timer
	s.reload = timer
}
func (s *Sequencer) update() bool {
	return s.timer == s.reload
}
func (p *Pulse) fired() bool {
	return p.sequencer.timer == p.sequencer.reload
}
func (s *Sequencer) tick() {
	s.clock++

	if s.timer > 0 {
		s.timer--
	} else {
		s.timer = s.reload
	}
}

type Pulse struct {
	pulseOne  bool // 1 if true else 2
	sequencer Sequencer

	dutyCycleMode  uint8 // 0,1,2,3
	dutyIndex      uint8 // 0..7
	lenCounterHalt bool  // 1 means go forever

	constVolume  bool // these two are         1
	envelopeFlag bool // opposites (same bit)  0

	volume       uint8 // in case of const volume
	envelopeDivP uint8 // same 8b as above but when envelope is set

	timer uint16

	lenCounterLoad uint8

	enabled bool

	apu *Apu

	clock uint64
}

func (p *Pulse) init(pulseOne bool, apu *Apu) {
	p.pulseOne = pulseOne
	p.enabled = false
	p.timer = 0
	p.dutyCycleMode = 0
	p.dutyIndex = 0
	p.apu = apu
}
func (p *Pulse) tick() {
	p.clock++
	p.sample()

	if p.enabled {
		p.sequencer.tick()
		if p.fired() {
			p.dutyIndex = (p.dutyIndex + 1) % 8
		}
	}
}

func (p *Pulse) dutyTable() [][]uint8 {
	return [][]uint8{
		{0, 1, 0, 0, 0, 0, 0, 0}, // 12.5%
		{0, 1, 1, 0, 0, 0, 0, 0}, // 25%
		{0, 1, 1, 1, 1, 0, 0, 0}, // 50%
		{1, 0, 0, 1, 1, 1, 1, 1}, // ~25%
	}
}

func (p *Pulse) sample() {
	output := p.dutyTable()[p.dutyCycleMode][p.dutyIndex]

	sampleRate := nesApuFrequency / p.apu.speaker.sampleRate
	if (p.clock % uint64(sampleRate)) == 0 {
		if output > 0 {
			p.apu.AddSample(nesApuVolumeGain * float64(p.volume))
		} else {
			p.apu.AddSample(0.0)
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
		a.pulse1.write8(addr, val)
	case addr >= 0x4004 && addr <= 0x4007:
		a.pulse2.write8(addr, val)
	}
}
func (p *Pulse) write8(addr uint16, val uint8) {
	if !p.pulseOne {
		addr -= 4
	}
	switch addr {
	// duty, len counter halt, const volume or envelope
	case 0x4000:
		p.dutyCycleMode = (val & 0xC0) >> 6
		if p.lenCounterHalt = true; (val & 0x20) == 0 {
			p.lenCounterHalt = false
		}
		if p.constVolume = true; (val & 0x10) == 0 {
			p.constVolume = false
		}
		p.volume = val & 0xF
		p.envelopeDivP = val & 0xF
		// sweep
	case 0x4001:
		//fmt.Printf("Sweep not supported!\n")
		// timer low
	case 0x4002:
		p.timer = (p.timer & 0xF0) | uint16(val)
		// timer high and len counter load
	case 0x4003:
		p.timer = uint16(val&0x7)<<8 | (p.timer & 0xFF)
		// The sequencer is immediately restarted at the first value of the
		// current sequence.
		// The envelope is also restarted. The period divider is not reset
		p.sequencer.set(p.timer)
		if val != 0 {
			p.enabled = true
		}
	}
}
