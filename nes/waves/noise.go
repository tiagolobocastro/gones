package waves

type Noise struct {
	constVolume bool  // these two are         1
	volume      uint8 // in case of const volume

	modeBit       uint8 // 6 if Mode flag is set, otherwise 1
	shiftRegister uint16

	timer    Timer
	duration DurationCounter
	envelope Envelope

	clock   uint64
	enabled bool
}

func (n *Noise) Init() {
	n.clock = 0
	n.modeBit = 1
	n.shiftRegister = 1
	n.duration.reset()
	n.envelope.reset()
	n.timer.reset()
	n.enabled = true
}
func (n *Noise) Tick() {
	n.clock++
	if n.timer.tick() {
		feedback := (n.shiftRegister & 0x1) ^ ((n.shiftRegister >> n.modeBit) & 0x1)
		n.shiftRegister >>= 1
		n.shiftRegister |= (feedback & 0x1) << 14
	}
}

func (n *Noise) noisePeriod() []uint16 {
	return []uint16{
		4, 8, 16, 32, 64, 96, 128, 160, 202,
		254, 380, 508, 762, 1016, 2034, 4068,
	}
}

func (n *Noise) Write8(addr uint16, val uint8) {
	switch addr {
	// duty, len counter halt, const volume or envelope
	case 0x400C:
		n.duration.set(!((val & 0x20) == 0))

		n.volume = val & 0xF
		if n.constVolume = true; (val & 0x10) == 0 {
			n.constVolume = false
			n.envelope.loop = n.duration.halt
			n.envelope.reload = n.volume
		}
	case 0x400E:
		n.timer.set(n.noisePeriod()[val&0xF])
		if (val & 0x80) != 0 {
			n.modeBit = 6
		} else {
			n.modeBit = 1
		}
	case 0x400F:
		n.duration.reload((val & 0xF8) >> 3)
		n.envelope.start = true
	}
}

func (n *Noise) Sample() float64 {
	if (n.shiftRegister&0x1) == 0 && !n.duration.mute() {
		if n.constVolume {
			return float64(n.volume)
		} else {
			return float64(n.envelope.decay)
		}
	} else {
		return 0.0
	}
}
func (n *Noise) Enabled() bool {
	return !n.duration.mute()
}
func (n *Noise) Enable(yes bool) {
	n.enabled = yes
	if !yes {
		n.duration.counter = 0
	}
}

func (n *Noise) QuarterFrameTick() {
	n.envelope.tick()
}
func (n *Noise) HalfFrameTick() {
	n.duration.tick()
}
