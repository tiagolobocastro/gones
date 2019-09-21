package gones

const (
	mapperNROM = iota
)

// atm the mirror prg is hardcoded which is wrong, that should be programmatic...

type Mapper struct {
	bus
	cart  *Cartridge
	mType byte
}

//CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
//CPU $8000-$BFFF: First 16 KB of ROM.
//CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
func (m *Mapper) read8(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return m.cart.chr.read8(addr)
	case addr < 0x8000:
		return m.cart.ram.read8(addr)
	case addr < 0xC000:
		return m.cart.prg.read8(uint16(int(addr) % m.cart.prg.size()))
	case addr <= 0xFFFF:
		return m.cart.prg.read8(uint16(int(addr) % m.cart.prg.size()))
	}
	return 0
}
func (m *Mapper) write8(addr uint16, val uint) {

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
		return m.nes.ppu.read8(addr /* % 8 */)

	case addr < 0x4018:
		return 0 // read from APU and I-O
	case addr < 0x4020:
		return 0 // APU

	case addr <= 0xFFFF:
		return m.nes.cart.mapper.read8(addr)
	}
	return 0
}

func (m *cpuMapper) write8(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		m.nes.ram.write8(addr%2048, val)

	case addr < 0x4000:
		m.nes.ppu.write8(addr%8, val)

	case addr < 0x4018:
		// APU and I-O
	case addr < 0x4020:
		// APU

		// case addr <= 0xFFFF: m.nes.cart.write
	}
}
