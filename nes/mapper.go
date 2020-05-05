package gones

const (
	mapperNROM  = iota
	mapperUnROM = 2
)

type Mapper interface {
	busInt
}

type MapperNROM struct {
	cart *Cartridge
}

//CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
//CPU $8000-$BFFF: First 16 KB of ROM.
//CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
func (m *MapperNROM) read8(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return m.cart.chr.read8(addr)
	case addr < 0x8000:
		return m.cart.ram.read8(addr)
	default:
		return m.cart.prg.read8(uint16(int(addr) % m.cart.prg.size()))
	}
}
func (m *MapperNROM) write8(addr uint16, val uint8) {
	panic("write not implemented!")
}

// CPU Mapping Table
// Address range 	Size 	Device
// $0000-$07FF 		$0800 	2KB internal RAM
// $0800-$0FFF 		$0800 	Mirrors of $0000-$07FF
// $1000-$17FF 		$0800
// $1800-$1FFF 		$0800
// $2000-$2007 		$0008 	NES PPU registers
// $2008-$3FFF 		$1FF8 	Mirrors of $2000-2007 (repeats every 8 bytes)
// $4000-$4017 		$0018 	NES APU and I/O registers
// $4018-$401F 		$0008 	APU and I/O functionality that is normally disabled. See CPU Test Mode.
// $4020-$FFFF 		$BFE0 	Cartridge space: PRG ROM, PRG RAM, and mapper registers (See Note)
type cpuMapper struct {
	*nes
}

func (m *cpuMapper) read8(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return m.nes.ram.read8(addr % 2048)

	case addr < 0x4000:
		return m.nes.ppu.read8(addr)

	case addr < 0x4016:
		// read from APU and I-O
		panic("address range not implemented!")
	case addr < 0x4018:
		// Controller
		return m.nes.ctrl.read8(addr)
	case addr < 0x4020:
		// APU
		panic("address range not implemented!")

	default:
		return m.nes.cart.mapper.read8(addr)
	}
	return 0
}

func (m *cpuMapper) write8(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		m.nes.ram.write8(addr%2048, val)

	case addr < 0x4000:
		m.nes.ppu.write8(addr, val)

	case addr < 0x4008:
		m.nes.apu.write8(addr, val)

	case addr == 0x4014:
		m.nes.dma.write8(addr, val)

	case addr < 0x4016:
		// I-O
		// panic("address range not implemented!")
	case addr < 0x4018:
		// Controller
		m.nes.ctrl.write8(addr, val)
	case addr < 0x4020:
		// APU
		//panic("address range not implemented!")

	default:
		panic("cannot write to cart!")
	}
}

// DMA
// entity that handles writes to OAMDMA register
// it reads from the cpu and copies to the ppu OAMDATA register

type dmaMapper struct {
	*nes
}

func (m *dmaMapper) read8(addr uint16) uint8 {
	// read from the cpu
	return m.nes.cpu.read8(addr)
}

func (m *dmaMapper) write8(addr uint16, val uint8) {
	// and copy to the ppu
	m.nes.ppu.write8(addr, val)
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
	case addr < 0x2000:
		return m.nes.cart.mapper.read8(addr)
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
	case addr < 0x4000:
		m.nes.ppu.palette.write8(addr%32, val)
	}
}

// APU
// Sound
// Could also be used to keep track of time?
// Does the apu need any bus access??
type apuMapper struct {
	*nes
}
