package waves

type Dmc struct {
	// Flags and Rate
	irqEnable bool
	loopFlag  bool
	rateTicks uint16

	directLoad uint8
	sampleAddr uint16
	sampleLen  uint16

	shiftRegister uint16

	timer Timer

	clock   uint64
	enabled bool
}

// https://wiki.nesdev.com/w/index.php/APU_DMC
// The rate determines for how many CPU cycles happen between changes in the
// output level during automatic delta-encoded sample playback. For example,
// on NTSC (1.789773 MHz), a rate of 428 gives a frequency of
// 1789773/428 Hz = 4181.71 Hz. These periods are all even numbers because
// there are 2 CPU cycles in an APU cycle. A rate of 428 means the output
// level changes every 214 APU cycles.
func rateTable() []uint16 {
	return []uint16{
		428, 380, 340, 320, 286, 254, 226, 214,
		190, 160, 142, 128, 106, 84, 72, 54,
	}
}
func sampleAddr(A uint8) uint16 {
	return 0xC000 + (uint16(A) * 64)
}
func sampleLen(L uint8) uint16 {
	return 1 + (uint16(L) * 16)
}

func (d *Dmc) Init() {
	d.irqEnable = false
	d.loopFlag = false
	d.rateTicks = rateTable()[0]
	d.directLoad = 0
	d.sampleAddr = sampleAddr(0)
	d.sampleLen = sampleLen(0)

	d.clock = 0
	d.shiftRegister = 1
	d.timer.reset()
	d.enabled = true
}

func (d *Dmc) Tick() {
	d.clock++
	if d.timer.tick() {
	}
}

func (d *Dmc) Write8(addr uint16, val uint8) {
	switch addr {
	// Flags and Rate
	case 0x4010:
		d.irqEnable = (val & 0x80) != 0
		d.loopFlag = (val & 0x40) != 0
		d.rateTicks = rateTable()[val&0xF]

		// Direct load
	case 0x4011:
		d.directLoad = val & 0x7F

		// Sample address
	case 0x4012:
		d.sampleAddr = sampleAddr(val)

		// Sample length
	case 0x4013:
		d.sampleLen = sampleLen(val)
	}
}

func (d *Dmc) Sample() float64 {
	if d.enabled {
		return 0.0
	}
	return 0.0
}
func (d *Dmc) Enabled() bool {
	//return d.remaining > 0
	return false
}
func (d *Dmc) Enable(yes bool) {
	d.enabled = yes
	if !yes {
		//d.duration.counter = 0
	}
}
