package mappers

import (
	"log"

	"github.com/tiagolobocastro/gones/lib/common"
)

type MapperMMC1 struct {
	cart *Cartridge

	// mapper mmc1 logic
	// https://wiki.nesdev.com/w/index.php/MMC1
	shift    uint8
	control  uint8
	chrBank0 uint8
	chrBank1 uint8
	prgBank  uint8

	mirror      uint8
	counter     uint8
	prgBankMode uint8
	chrBankMode uint8

	prgBanks [2]uint32
	chrBanks [2]uint16
}

func (m *MapperMMC1) Tick() {}

func (m *MapperMMC1) Init() {
	m.writeInner(0x8000, 0x1F)
}

// 7  bit  0
// ---- ----
// Rxxx xxxD
// |       |
// |       +- Data bit to be shifted into shift register, LSB first
// +--------- 1: Reset shift register and write Control with (Control OR $0C),
//              locking PRG ROM at $C000-$FFFF to the last bank.
func (m *MapperMMC1) writeLoad(addr uint16, val uint8) {
	if (val & 0x80) != 0 {
		m.shift = 0x0
		m.counter = 0
	} else {
		m.shift = m.shift | (val&0x1)<<m.counter
		m.counter++

		if m.counter == 5 {

			m.writeInner(addr, m.shift)

			m.shift = 0x0
			m.counter = 0
		}
	}
}

func (m *MapperMMC1) writeInner(addr uint16, val uint8) {
	switch {
	case addr >= 0x8000 && addr <= 0x9FFF:
		m.writeControl(val)
	case addr >= 0xA000 && addr <= 0xBFFF:
		m.writeCHRBank0(val)
	case addr >= 0xC000 && addr <= 0xDFFF:
		m.writeCHRBank1(val)
	case addr >= 0xE000:
		m.writePRGBank(val)
	}
	m.updateAllBanks()
}

// Control (internal, $8000-$9FFF)
// 4bit0
// -----
// CPPMM
// |||||
// |||++- Mirroring (0: one-screen, lower bank; 1: one-screen, upper bank;
// |||               2: vertical; 3: horizontal)
// |++--- PRG ROM bank mode (0, 1: switch 32 KB at $8000, ignoring low bit of bank number;
// |                         2: fix first bank at $8000 and switch 16 KB bank at $C000;
// |                         3: fix last bank at $C000 and switch 16 KB bank at $8000)
// +----- CHR ROM bank mode (0: switch 8 KB at a time; 1: switch two separate 4 KB banks)
func (m *MapperMMC1) writeControl(val uint8) {
	m.mirror = val & 0x3
	switch m.mirror {
	case 0:
		m.cart.SetMirroring(common.SingleScreenMirroring)
	case 1:
		m.cart.SetMirroring(common.SingleScreenMirroring)
	case 2:
		m.cart.SetMirroring(common.VerticalMirroring)
	case 3:
		m.cart.SetMirroring(common.HorizontalMirroring)
	}
	m.prgBankMode = (val >> 2) & 0x3
	m.chrBankMode = val >> 4
}
func (m *MapperMMC1) updateAllBanks() {
	m.updateCHRBank0()
	m.updateCHRBank1()
	m.updatePRGBank()
}

// CHR bank 0 (internal, $A000-$BFFF)
//
// 4bit0
// -----
// CCCCC
// |||||
// +++++- Select 4 KB or 8 KB CHR bank at PPU $0000 (low bit ignored in 8 KB mode)
func (m *MapperMMC1) writeCHRBank0(val uint8) {
	m.chrBank0 = val & 0x1f
}
func (m *MapperMMC1) updateCHRBank0() {
	switch m.chrBankMode {
	case 0:
		// 8 KB
		bank := (uint16(m.chrBank0) >> 1) * 0x2000
		m.chrBanks[0] = bank
		m.chrBanks[1] = bank + 0x1000
	case 1:
		// 4 KB
		bank := uint16(m.chrBank0) * 0x1000
		m.chrBanks[0] = bank
	}
}

// CHR bank 1 (internal, $C000-$DFFF)
//
// 4bit0
// -----
// CCCCC
// |||||
// +++++- Select 4 KB CHR bank at PPU $1000 (ignored in 8 KB mode)
func (m *MapperMMC1) writeCHRBank1(val uint8) {
	m.chrBank1 = val & 0x1f
}
func (m *MapperMMC1) updateCHRBank1() {
	switch m.chrBankMode {
	case 0:
		// 8 KB
		// noop
	case 1:
		// 4 KB
		bank := uint16(m.chrBank1) * 0x1000
		m.chrBanks[1] = bank
	}
}

// PRG bank (internal, $E000-$FFFF)
//
// 4bit0
// -----
// RPPPP
// |||||
// |++++- Select 16 KB PRG ROM bank (low bit ignored in 32 KB mode)
// +----- PRG RAM chip enable (0: enabled; 1: disabled; ignored on MMC1A)
func (m *MapperMMC1) writePRGBank(val uint8) {
	m.prgBank = val
	m.updatePRGBank()
}
func (m *MapperMMC1) updatePRGBank() {
	switch m.prgBankMode {
	case 0, 1:
		// 32KB mode
		bankV := uint32(m.prgBank) >> 1
		bank := 0x8000 * bankV

		m.prgBanks[0] = bank
		m.prgBanks[1] = bank + 0x4000
	case 2:
		//  2: fix first bank at $8000 and switch 16 KB bank at $C000;
		m.prgBanks[0] = 0x0000
		m.prgBanks[1] = 0x4000 * uint32(m.prgBank)
	case 3:
		// 3: fix last bank at $C000 and switch 16 KB bank at $8000)
		m.prgBanks[0] = 0x4000 * uint32(m.prgBank)
		m.prgBanks[1] = uint32(m.cart.prgRom.Size()) - 0x4000
	}
}

// CPU $6000-$7FFF: 8 KB PRG RAM bank, (optional)
// CPU $8000-$BFFF: 16 KB PRG ROM bank, either switchable or fixed to the first bank
// CPU $C000-$FFFF: 16 KB PRG ROM bank, either fixed to the last bank or switchable
// PPU $0000-$0FFF: 4 KB switchable CHR bank
// PPU $1000-$1FFF: 4 KB switchable CHR bank
func (m *MapperMMC1) Read8(addr uint16) uint8 {
	switch {
	// PPU - normally mapped by the cartridge to a CHR-ROM or CHR-RAM,
	// often with a bank switching mechanism.
	case addr < 0x1000:
		return m.cart.chr.Read8(addr + m.chrBanks[0])
	case addr < 0x2000:
		return m.cart.chr.Read8(addr - 0x1000 + m.chrBanks[1])
	case addr >= 0x6000 && addr < 0x8000:
		return m.cart.prgRam.Read8(addr - 0x6000)
	case addr >= 0x8000 && addr < 0xC000:
		offset := uint32(addr - 0x8000)
		return m.cart.prgRom.Read8w(m.prgBanks[0] + offset)
	case addr >= 0xC000:
		offset := uint32(addr - 0xC000)
		return m.cart.prgRom.Read8w(m.prgBanks[1] + offset)
	default:
		log.Panicf("read not implemented for 0x%04x!", addr)
		return 0
	}
}
func (m *MapperMMC1) Write8(addr uint16, val uint8) {
	switch {
	case addr < 0x1000:
		m.cart.chr.Write8(addr+m.chrBanks[0], val)
	case addr < 0x2000:
		m.cart.chr.Write8(addr-0x1000+m.chrBanks[1], val)
	case addr >= 0x6000 && addr < 0x8000:
		m.cart.prgRam.Write8(addr-0x6000, val)
	case addr >= 0x8000:
		m.writeLoad(addr, val)
	default:
		log.Panicf("write not implemented for 0x%04x!", addr)
	}
}

func (m *MapperMMC1) Serialise(s common.Serialiser) error {
	return s.Serialise(
		m.shift, m.control, m.chrBank0, m.chrBank1, m.prgBank, m.mirror,
		m.counter, m.prgBankMode, m.chrBankMode, m.prgBanks, m.chrBanks,
	)
}
func (m *MapperMMC1) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&m.shift, &m.control, &m.chrBank0, &m.chrBank1, &m.prgBank, &m.mirror,
		&m.counter, &m.prgBankMode, &m.chrBankMode, &m.prgBanks, &m.chrBanks,
	)
}
