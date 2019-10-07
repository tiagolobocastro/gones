package gones

type Ppu struct {
	busInt

	cycle    int
	scanLine int
	verbose  bool

	// cpu mapper
	regs [9]register

	// internal registers: http://wiki.nesdev.com/w/index.php/PPU_scrolling
	vRAM    register16 // Current VRAM address (15 bits)
	tRAM    register16 // Temporary VRAM address (15 bits); can also be thought of as the address of the top left onscreen tile.
	xFine   register   // Fine X scroll (3 bits)
	wToggle register   // First or second write toggle (1 bit)

	palette ppuPalette

	interrupts iInterrupt
}

func (p *Ppu) init(busInt busInt, verbose bool, interrupts iInterrupt) {
	p.verbose = verbose
	p.busInt = busInt
	p.interrupts = interrupts
	p.cycle = 0
	p.scanLine = 0

	p.vRAM.init("v", 0)
	p.tRAM.init("t", 0)
	p.xFine.init("x", 0)
	p.wToggle.init("w", 0)
}

func (p *Ppu) reset() {
	p.init(p.busInt, p.verbose, p.interrupts)
}

// PPU Mapping Table
// Address range 	Size 	Device
// $0000-$0FFF 		$1000 	Pattern table 0
// $1000-$1FFF 		$1000 	Pattern table 1
// $2000-$23FF 		$0400 	Nametable 0
// $2400-$27FF 		$0400 	Nametable 1
// $2800-$2BFF 		$0400 	Nametable 2
// $2C00-$2FFF 		$0400 	Nametable 3
// $3000-$3EFF 		$0F00 	Mirrors of $2000-$2EFF
// $3F00-$3F1F 		$0020 	Palette RAM indexes
// $3F20-$3FFF 		$00E0 	Mirrors of $3F00-$3F1F
type ppuMapper struct {
	*nes
}

func (m *ppuMapper) read8(addr uint16) uint8 {
	switch {
	// PPU VRAM or controlled via the Cartridge Mapper
	case addr < 0x3000:
		return m.nes.vRam.read8(addr % 2048)
	case addr < 0x3F00:
		return m.nes.vRam.read8(addr % 2048)

	// internal palette control
	case addr < 0x3F20:
		return m.nes.ppu.palette.read8(addr % 32)
	case addr < 0x4000:
		return m.nes.ppu.palette.read8(addr % 32)
	}
	return 0
}

func (m *ppuMapper) write8(addr uint16, val uint8) {
	switch {
	// PPU VRAM or controlled via the Cartridge Mapper
	case addr < 0x3000:
		m.nes.vRam.write8(addr%2048, val)
	case addr < 0x3F00:
		m.nes.vRam.write8(addr%2048, val)

	// internal palette control
	case addr < 0x3F20:
		m.nes.ppu.palette.write8(addr%32, val)
	case addr < 0x4000:
		m.nes.ppu.palette.write8(addr%32, val)
	}
}

// interrupt
// only look at the CPU NMI for now
// need to implement the interrupt delay as well since the cpu and ppu and not on the same clock
func (p *Ppu) raise(flag uint8) {
	if (flag & cpuIntNMI) != 0 {
		p.regs[PPUSTATUS].val |= 0x80

		if p.getNMIVertical() == 1 {
			p.interrupts.raise(flag & cpuIntNMI)
		}
	}
}
func (p *Ppu) clear(flag uint8) {
	if (flag & cpuIntNMI) != 0 {
		p.regs[PPUSTATUS].val &= 0x7F
		p.interrupts.clear(flag & cpuIntNMI)
	}
}

func (p *Ppu) tick() {
	p.cycle += 1

	if p.cycle > 340 {
		p.scanLine += 1
		p.cycle = 0

		if p.scanLine == 241 {
			p.raise(cpuIntNMI)
		}

		if p.scanLine > 260 {
			p.scanLine = 0
			// may already be cleared as reading from PPSTATUS will do so
			p.clear(cpuIntNMI)
		}
	}
}

func (p *Ppu) clock() {

	// 3 ppu ticks per 1 cpu
	p.exec()
	p.exec()
	p.exec()
}

func (p *Ppu) exec() {

	// first do the work, and only then tick?

	// let's add a simple sprite display or something like that
	// so let's do the bare minimum for the ppu setup

	p.tick()
}

// cpu can read from the ppu through the control registers

// BusInt
func (p *Ppu) read8(addr uint16) uint8 {
	if addr < 0x4000 {
		// incomplete decoding means 0x2000-0x2007 are mirrored every 8 bytes
		addr = 0x2000 + addr%8
	}

	switch addr {
	// PPU Status (PPUSTATUS) - RDONLY
	case 0x2002:
		return p.getSTATUS()
	// PPU OAM Data (OAMDATA)
	case 0x2004:
		return p.regs[OAMDATA].read()
	// PPU Data (PPUDATA)
	case 0x2007:
		return p.regs[PPUDATA].read()
	}

	return 0
}
func (p *Ppu) write8(addr uint16, val uint8) {
	if addr < 0x4000 {
		// incomplete decoding means 0x2000-0x2007 are mirrored every 8 bytes
		addr = 0x2000 + addr%8
	}

	switch addr {
	// PPU Control (PPUCTRL) - WRONLY
	case 0x2000:
		p.regs[PPUCTRL].write(val)
	// PPU Mask (PPUMASK) - WRONLY
	case 0x2001:
		p.regs[PPUMASK].write(val)
	// PPU OAM Data (OAMADDR) - WRONLY
	case 0x2003:
		p.regs[OAMADDR].write(val)
	// PPU OAM Data (OAMDATA)
	case 0x2004:
		p.regs[OAMDATA].write(val)
	// PPU Scrolling (PPUSCROLL) - WRONLY
	case 0x2005:
		p.regs[PPUSCROLL].write(val)
	// PPU Address (PPUADDR) - WRONLY
	case 0x2006:
		p.writePPUAddr(val)
	// PPU Data (PPUDATA)
	case 0x2007:
		p.writePPUData(val)
	// PPU OAM DMA (OAMDMA) - WRONLY
	case 0x4014:
		p.regs[OAMDMA].write(val)
	}
}
