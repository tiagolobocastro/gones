package mappers

import (
	"fmt"
)

type MapperNROM struct {
	cart *Cartridge
}

func (m *MapperNROM) Init() {}

//CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
//CPU $8000-$BFFF: First 16 KB of ROM.
//CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
func (m *MapperNROM) Read8(addr uint16) uint8 {
	switch {
	// PPU - normally mapped by the cartridge to a CHR-ROM or CHR-RAM,
	// often with a bank switching mechanism.
	case addr < 0x2000:
		return m.cart.chr.Read8(addr)
	case addr < 0x8000:
		return m.cart.prgRam.Read8(addr)
	default:
		return m.cart.prgRom.Read8(uint16(int(addr) % m.cart.prgRom.Size()))
	}
}
func (m *MapperNROM) Write8(addr uint16, val uint8) {
	switch {
	case addr >= 0x6000 && addr <= 0x8000:
		m.cart.prgRam.Write8(addr%0x6000, val)
	default:
		panic(fmt.Sprintf("write not implemented for %v!", addr))
	}
}
