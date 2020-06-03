package gones

import "github.com/tiagolobocastro/gones/nes/common"

// BusInt
type dma struct {
	common.BusInt

	clock uint

	// dma nBytes from cpu to ppu
	nBytes uint16

	byteRd  uint8
	cpuAddr uint16
	ppuAddr uint16

	delay bool
}

func (d *dma) init(busInt common.BusInt) {
	d.BusInt = busInt
	d.nBytes = 0
}
func (d *dma) reset() {
	d.init(d.BusInt)
}

func (d *dma) active() bool {
	return d.nBytes > 0
}

func (d *dma) ticks(nTicks int) {

	for i := 0; i < nTicks; i++ {
		d.tick()
	}
}

func (d *dma) tick() {

	// clock required for the delay logic
	d.clock++
	d.exec()
}

func (d *dma) exec() {

	if d.nBytes > 0 {

		// dma transfer starts on the next even clock cycle
		if d.delay {
			if d.clock%2 == 1 {
				d.delay = false
			}
		} else {
			if d.clock%2 == 0 {

				d.byteRd = d.BusInt.Read8(d.cpuAddr)
				d.cpuAddr++

			} else {

				d.BusInt.Write8(d.ppuAddr, d.byteRd)
				d.nBytes--
			}
		}
	} else {
		d.delay = true
	}
}

func (d *dma) setupTransfer(cpuAddr uint16) {
	d.cpuAddr = cpuAddr
	d.ppuAddr = 0x2004 // OAMDATA
	d.nBytes = 256
}

func (d *dma) Read8(addr uint16) uint8 {
	return 0
}

func (d *dma) Write8(addr uint16, val uint8) {
	switch addr {
	case 0x4014:
		d.setupTransfer(uint16(val) << 8)
	}
}
