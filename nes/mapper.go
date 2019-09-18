package gones

const (
	mapperNROM = iota
)

// atm the mirror prg is hardcoded which is wrong, that should be programmatic...

type Mapper struct {
	bus
	cart  *Cartridge
	mType byte

	devEntries []busMemMapEntry
}

func newMapper(cart *Cartridge, mapperId byte) *Mapper {

	m := &Mapper{cart: cart, mType: mapperId}
	m.devEntries = m.getMapperEntries()

	return m
}

func (m *Mapper) getMapperEntries() []busMemMapEntry {
	switch m.mType {
	case mapperNROM:
		return []busMemMapEntry{
			//CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
			//CPU $8000-$BFFF: First 16 KB of ROM.
			//CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
			// need to sort out the rom mirror...
			{busAddrRange{addrRange{0x0000, 0x1FFF, 0}, 0}, m.cart.chr},
			{busAddrRange{addrRange{0x6000, 0x7FFF, 0}, 0}, m.cart.ram},
			{busAddrRange{addrRange{0x8000, 0xBFFF, 0}, 0}, m.cart.prg},
			{busAddrRange{addrRange{0xC000, 0xFFFF, 0}, 0}, m.cart.prg},
		}
	}

	return nil
}
