package waves

type Pulse struct {
	dutyCycleMode  uint8 // 0,1,2,3
	dutyIndex      uint8 // 0..7
	lenCounterHalt bool  // 1 means go forever

	constVolume  bool // these two are         1
	envelopeFlag bool // opposites (same bit)  0

	volume       uint8 // in case of const volume
	envelopeDivP uint8 // same 8b as above but when envelope is set

	lenCounterLoad uint8
	lenCounter     uint8

	pulseOne bool // 1 if true else 2

	sequencer Sequencer
	duration  DurationCounter

	clock uint64
}

func (p *Pulse) Init(pulseOne bool) {
	p.pulseOne = pulseOne
	p.dutyCycleMode = 0
	p.dutyIndex = 0
	p.clock = 0
	p.duration.reset()
	p.sequencer.reset()
}
func (p *Pulse) Tick() {
	p.clock++

	if p.sequencer.tick() {
		p.dutyIndex = (p.dutyIndex + 1) % 8
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

func (p *Pulse) Write8(addr uint16, val uint8) {
	if !p.pulseOne {
		addr -= 4
	}
	switch addr {
	// duty, len counter halt, const volume or envelope
	case 0x4000:
		p.dutyCycleMode = (val & 0xC0) >> 6
		p.duration.set(!((val & 0x20) == 0))
		if p.constVolume = true; (val & 0x10) == 0 {
			p.constVolume = false
		}
		p.volume = val & 0xF
		p.envelopeDivP = val & 0xF
		// sweep
	case 0x4001:
		//fmt.Printf("Sweep not supported!\n")
	case 0x4002:
		p.sequencer.resetLow(val)
	case 0x4003:
		// The sequencer is immediately restarted at the first value of the
		// current sequence.
		p.sequencer.resetHigh(val & 0x7)
		p.dutyIndex = 0

		// The envelope is also restarted. The period divider is not reset
		p.duration.reload((val & 0xF8) >> 3)
	}
}

func (p *Pulse) Sample() float64 {
	output := p.dutyTable()[p.dutyCycleMode][p.dutyIndex]

	// since we can't perfectly achieve the right sampling freq
	// maybe let's try using a more "perfect" sampling freq
	// and then using a filter

	if p.duration.counter != 0 && output > 0 &&
		p.sequencer.reload >= 8 && p.sequencer.reload < 0x7FF {
		return float64(p.volume)
	} else {
		return 0.0
	}
}

func (p *Pulse) QuarterFrameTick() {
	p.envelopeTick()
}
func (p *Pulse) HalfFrameTick() {
	p.duration.tick()
	p.sweepUnitTick()
}

func (p *Pulse) sweepUnitTick() {
}
func (p *Pulse) envelopeTick() {
}
