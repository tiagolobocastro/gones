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

func newMapper(cart *Cartridge, mapperId byte) *Mapper {

	m := &Mapper{cart: cart, mType: mapperId}

	return m
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
		return m.nes.ppu.read8(addr % 8)

	case addr < 0x4018:
		return 0 // read from APU and I-O
	case addr < 0x4020:
		return 0 // APU

		// case addr <= 0xFFFF: return m.nes.cart.rea
	}
	return 0
}
