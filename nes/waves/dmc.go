package waves

import "github.com/tiagolobocastro/gones/nes/common"

type Dmc struct {
	common.BusInt

	// Flags and Rate
	irqEnable bool
	loopFlag  bool
	rateTicks uint16

	outputLevel   uint8
	sampleAddrRld uint16
	sampleLenRld  uint16

	sampleBuffer uint8
	sampleReady  bool
	sampleAddr   uint16
	sampleLen    uint16

	shiftRegister uint8
	bitsRemaining uint8
	silenceFlag   bool

	timer Timer

	clock   uint64
	enabled bool
}

func (d *Dmc) Serialise(s common.Serialiser) error {
	return s.Serialise(
		d.irqEnable, d.loopFlag, d.rateTicks, d.outputLevel, d.sampleAddrRld,
		d.sampleLenRld, d.sampleBuffer, d.sampleReady, d.sampleAddr, d.sampleLen,
		d.shiftRegister, d.bitsRemaining, d.silenceFlag, d.clock, d.enabled,
	)
}
func (d *Dmc) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&d.irqEnable, &d.loopFlag, &d.rateTicks, &d.outputLevel, &d.sampleAddrRld,
		&d.sampleLenRld, &d.sampleBuffer, &d.sampleReady, &d.sampleAddr, &d.sampleLen,
		&d.shiftRegister, &d.bitsRemaining, &d.silenceFlag, &d.clock, &d.enabled,
	)
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

func (d *Dmc) Init(busInt common.BusInt) {
	d.BusInt = busInt

	d.irqEnable = false
	d.loopFlag = false
	d.rateTicks = rateTable()[0] / 2
	d.outputLevel = 0
	d.sampleAddrRld = sampleAddr(0)
	d.sampleAddr = d.sampleAddrRld
	d.sampleLenRld = sampleLen(0)
	d.sampleLen = d.sampleAddrRld
	d.sampleBuffer = 0
	d.sampleReady = false

	d.clock = 0
	d.shiftRegister = 1
	d.bitsRemaining = 0
	d.silenceFlag = false
	d.timer.reset()
	d.enabled = false
}
func (d *Dmc) reload() {
	if d.sampleLen == 0 {
		d.sampleLen = d.sampleLenRld
		d.sampleAddr = d.sampleAddrRld
	}
}

func (d *Dmc) Tick() {
	d.clock++
	if d.timer.tick() {
		if !d.sampleReady && d.sampleLen > 0 {
			d.sampleBuffer = d.Read8(d.sampleAddr)
			d.sampleAddr++
			d.sampleLen--
			if d.sampleAddr == 0 {
				d.sampleAddr = 0x8000
			}
			// todo: stall cpu for a few cycles:
			// https://wiki.nesdev.com/w/index.php/APU_DMC
			if d.loopFlag {
				d.reload()
			}
		}

		if d.bitsRemaining == 0 {
			d.bitsRemaining = 8

			if !d.sampleReady {
				d.silenceFlag = true
			} else {
				d.silenceFlag = false
				d.shiftRegister = d.sampleBuffer
			}
		}

		if !d.silenceFlag {
			if (d.shiftRegister & 1) == 1 {
				if d.outputLevel <= 125 {
					d.outputLevel += 2
				}
			} else {
				if d.outputLevel >= 2 {
					d.outputLevel -= 2
				}
			}
			d.shiftRegister >>= 1
		}
		d.bitsRemaining--
	}
}

func (d *Dmc) Write8(addr uint16, val uint8) {
	switch addr {
	// Flags and Rate
	case 0x4010:
		d.irqEnable = (val & 0x80) != 0
		d.loopFlag = (val & 0x40) != 0
		d.rateTicks = rateTable()[val&0xF] / 2

		// Direct load
	case 0x4011:
		d.outputLevel = val & 0x7F

		// Sample address
	case 0x4012:
		d.sampleAddrRld = sampleAddr(val)

		// Sample length
	case 0x4013:
		d.sampleLenRld = sampleLen(val)
	}
}

func (d *Dmc) Sample() float64 {
	return float64(d.outputLevel)
	if d.enabled && !d.silenceFlag {
		return float64(d.outputLevel)
	}
	return 0.0
}
func (d *Dmc) Enabled() bool {
	return d.sampleLen > 0
}
func (d *Dmc) Enable(yes bool) {
	d.enabled = yes
	if !yes {
		d.sampleLen = 0
	} else {
		d.reload()
	}
}
