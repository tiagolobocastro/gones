package common

// BusInt
type Dma struct {
	BusInt

	clock uint

	// dma nBytes from cpu to ppu
	nBytes uint16

	byteRd  uint8
	cpuAddr uint16
	ppuAddr uint16

	delay bool
}

func (d *Dma) Serialise(s Serialiser) error {
	return s.Serialise(d.clock, d.nBytes, d.byteRd, d.cpuAddr, d.ppuAddr, d.delay)
}
func (d *Dma) DeSerialise(s Serialiser) error {
	return s.DeSerialise(&d.clock, &d.nBytes, &d.byteRd, &d.cpuAddr, &d.ppuAddr, &d.delay)
}

func (d *Dma) Init(busInt BusInt) {
	d.BusInt = busInt
	d.nBytes = 0
}
func (d *Dma) Reset() {
	d.Init(d.BusInt)
}

func (d *Dma) Active() bool {
	return d.nBytes > 0
}

func (d *Dma) Ticks(nTicks int) {

	for i := 0; i < nTicks; i++ {
		d.tick()
	}
}

func (d *Dma) tick() {

	// clock required for the delay logic
	d.clock++
	d.exec()
}

func (d *Dma) exec() {

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

func (d *Dma) setupTransfer(cpuAddr uint16) {
	d.cpuAddr = cpuAddr
	d.ppuAddr = 0x2004 // OAMDATA
	d.nBytes = 256
}

func (d *Dma) Read8(addr uint16) uint8 {
	return 0
}

func (d *Dma) Write8(addr uint16, val uint8) {
	switch addr {
	case 0x4014:
		d.setupTransfer(uint16(val) << 8)
	}
}
