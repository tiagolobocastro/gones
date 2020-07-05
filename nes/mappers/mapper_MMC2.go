package mappers

import (
	"fmt"
	"github.com/tiagolobocastro/gones/nes/common"
)

type MapperMMC2 struct {
	cart *Cartridge

	chrBankD0 uint8
	chrBankE0 uint8
	chrBankD1 uint8
	chrBankE1 uint8
	prgBank   uint8

	mirror uint8

	prgBanks [1]uint32
	chrBanks [4]uint32

	latch [2]uint8
}

func (m *MapperMMC2) Step() {}

func (m *MapperMMC2) Init() {
	m.latch[0] = 0xFD
	m.latch[1] = 0xFD
	m.mirror = m.cart.config.mirror
	m.chrBankD0 = 0
	m.chrBankE0 = 0
	m.chrBankD1 = 0
	m.chrBankE1 = 0
	m.prgBank = 0
}

func (m *MapperMMC2) writeInner(addr uint16, val uint8) {
	switch {
	case addr >= 0xA000 && addr <= 0xAFFF:
		m.writePRGBank(val)
	case addr >= 0xB000 && addr <= 0xBFFF:
		m.writeCHRBankD0(val)
	case addr >= 0xC000 && addr <= 0xCFFF:
		m.writeCHRBankE0(val)
	case addr >= 0xD000 && addr <= 0xDFFF:
		m.writeCHRBankD1(val)
	case addr >= 0xE000 && addr <= 0xEFFF:
		m.writeCHRBankE1(val)
	case addr >= 0xF000:
		m.writeMirroring(val)
	}
	m.updateAllBanks()
}

func (m *MapperMMC2) updateAllBanks() {
	m.updatePRGBank()
	m.updateCHRBanks()
}

func (m *MapperMMC2) updateCHRBanks() {
	// D0
	m.chrBanks[0] = 0x1000 * uint32(m.chrBankD0)
	// E0
	m.chrBanks[1] = 0x1000 * uint32(m.chrBankE0)
	// D1
	m.chrBanks[2] = 0x1000 * uint32(m.chrBankD1)
	// E1
	m.chrBanks[3] = 0x1000 * uint32(m.chrBankE1)
}

// PRG ROM bank select ($A000-$AFFF)
//
// 7  bit  0
// ---- ----
// xxxx PPPP
//      ||||
//      ++++- Select 8 KB PRG ROM bank for CPU $8000-$9FFF
func (m *MapperMMC2) writePRGBank(val uint8) {
	m.prgBank = val & 0xF
}
func (m *MapperMMC2) updatePRGBank() {
	m.prgBanks[0] = 0x2000 * uint32(m.prgBank)
}

// CHR ROM $FD/0000 bank select ($B000-$BFFF)
//
// 7  bit  0
// ---- ----
// xxxC CCCC
//    | ||||
//    +-++++- Select 4 KB CHR ROM bank for PPU $0000-$0FFF
//            used when latch 0 = $FD
func (m *MapperMMC2) writeCHRBankD0(val uint8) {
	m.chrBankD0 = val & 0x1f
	m.latch[0] = 0
}

// CHR ROM $FE/0000 bank select ($C000-$CFFF)
//
// 7  bit  0
// ---- ----
// xxxC CCCC
//    | ||||
//    +-++++- Select 4 KB CHR ROM bank for PPU $0000-$0FFF
//            used when latch 0 = $FE
func (m *MapperMMC2) writeCHRBankE0(val uint8) {
	m.chrBankE0 = val & 0x1f
	m.latch[0] = 0
}

// CHR ROM $FD/1000 bank select ($D000-$DFFF)
//
// 7  bit  0
// ---- ----
// xxxC CCCC
//    | ||||
//    +-++++- Select 4 KB CHR ROM bank for PPU $1000-$1FFF
//            used when latch 1 = $FD
func (m *MapperMMC2) writeCHRBankD1(val uint8) {
	m.chrBankD1 = val & 0x1f
}

// CHR ROM $FE/1000 bank select ($E000-$EFFF)
//
// 7  bit  0
// ---- ----
// xxxC CCCC
//    | ||||
//    +-++++- Select 4 KB CHR ROM bank for PPU $1000-$1FFF
//            used when latch 1 = $FE
func (m *MapperMMC2) writeCHRBankE1(val uint8) {
	m.chrBankE1 = val & 0x1f
}

// Mirroring ($F000-$FFFF)
//
// 7  bit  0
// ---- ----
// xxxx xxxM
//         |
//         +- Select nametable mirroring (0: vertical; 1: horizontal)
func (m *MapperMMC2) writeMirroring(val uint8) {
	m.mirror = val & 0x3
	switch m.mirror {
	case 0:
		m.cart.SetMirroring(common.VerticalMirroring)
	case 1:
		m.cart.SetMirroring(common.HorizontalMirroring)
	}
}

// PPU $0000-$0FFF: Two 4 KB switchable CHR ROM banks
// PPU $1000-$1FFF: Two 4 KB switchable CHR ROM banks
// CPU $6000-$7FFF: 8 KB PRG RAM bank (PlayChoice version only; contains a 6264 and 74139)
// CPU $8000-$9FFF: 8 KB switchable PRG ROM bank
// CPU $A000-$FFFF: Three 8 KB PRG ROM banks, fixed to the last three banks
func (m *MapperMMC2) Read8(addr uint16) uint8 {
	switch {
	case addr < 0x1000:
		v := uint8(0)
		if m.latch[0] == 0xFD {
			v = m.cart.chr.Read8w(uint32(addr) + m.chrBanks[0])
		} else {
			v = m.cart.chr.Read8w(uint32(addr) + m.chrBanks[1])
		}
		if addr == 0xFD8 {
			m.latch[0] = 0xFD
		} else if addr == 0xFE8 {
			m.latch[0] = 0xFE
		}
		return v
	case addr < 0x2000:
		v := uint8(0)
		if m.latch[1] == 0xFD {
			v = m.cart.chr.Read8w(uint32(addr-0x1000) + m.chrBanks[2])
		} else {
			v = m.cart.chr.Read8w(uint32(addr-0x1000) + m.chrBanks[3])
		}
		if addr >= 0x1FD8 && addr <= 0x1FDF {
			m.latch[1] = 0xFD
		} else if addr >= 0x1FE8 && addr <= 0x1FEF {
			m.latch[1] = 0xFE
		}
		return v

	case addr >= 0x6000 && addr < 0x8000:
		return m.cart.prgRam.Read8(addr - 0x6000)
	case addr >= 0x8000 && addr < 0xA000:
		return m.cart.prgRom.Read8w(uint32(addr-0x8000) + m.prgBanks[0])
	case addr >= 0xA000:
		offset := uint32(addr - 0xA000)
		return m.cart.prgRom.Read8w(offset + uint32(m.cart.prgRom.Size()) - 0x2000*3)
	default:
		panic(fmt.Sprintf("read not implemented for 0x%04x!", addr))
	}
}
func (m *MapperMMC2) Write8(addr uint16, val uint8) {
	switch {
	case addr < 0x1000:
		if m.latch[0] == 0xFD {
			m.cart.chr.Write8w(uint32(addr)+m.chrBanks[0], val)
		} else {
			m.cart.chr.Write8w(uint32(addr)+m.chrBanks[1], val)
		}
	case addr < 0x2000:
		if m.latch[1] == 0xFD {
			m.cart.chr.Write8w(uint32(addr-0x1000)+m.chrBanks[2], val)
		} else {
			m.cart.chr.Write8w(uint32(addr-0x1000)+m.chrBanks[3], val)
		}

	case addr >= 0x6000 && addr < 0x8000:
		m.cart.prgRam.Write8(addr-0x6000, val)
	case addr >= 0xA000:
		m.writeInner(addr, val)
	default:
		panic(fmt.Sprintf("write not implemented for 0x%04x!", addr))
	}
}

func (m *MapperMMC2) Serialise(s common.Serialiser) error {
	return s.Serialise(
		m.chrBankD0, m.chrBankE0, m.chrBankD1, m.chrBankE1, m.prgBank, m.mirror,
		m.prgBanks, m.chrBanks, m.chrBanks, m.latch,
	)
}
func (m *MapperMMC2) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&m.chrBankD0, &m.chrBankE0, &m.chrBankD1, &m.chrBankE1, &m.prgBank, &m.mirror,
		&m.prgBanks, &m.chrBanks, &m.chrBanks, &m.latch,
	)
}
