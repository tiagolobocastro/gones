package mappers

import (
	"fmt"
	"github.com/tiagolobocastro/gones/lib/common"
)

type MapperMMC3 struct {
	cart *Cartridge

	bankMode      uint8
	bankData      uint8
	prgRamProtect uint8
	irqLatch      uint8
	irqReload     uint8
	irqDisable    bool
	mirror        uint8
	registers     [8]uint8

	prgBanks [4]uint32
	chrBanks [8]uint32
}

func (m *MapperMMC3) Tick() {

}

func (m *MapperMMC3) Init() {
	m.mirror = m.cart.config.mirror
	m.updateAllBanks()
}

// The MMC3 has 4 pairs of registers at $8000-$9FFF, $A000-$BFFF, $C000-$DFFF, and $E000-$FFFF
// even addresses ($8000, $8002, etc.) select the low register and
// odd addresses ($8001, $8003, etc.) select the high register in each pair.
// These can be broken into two independent functional units: memory mapping ($8000, $8001, $A000, $A001) and
// scanline counting ($C000, $C001, $E000, $E001).
func (m *MapperMMC3) writeInner(addr uint16, val uint8) {
	even := (addr & 1) == 0
	odd := !even
	switch {
	case addr >= 0x8000 && addr <= 0x9FFF && even:
		m.writeBankSelect(val)
	case addr >= 0x8000 && addr <= 0x9FFF && odd:
		m.writeBankData(val)
	case addr >= 0xA000 && addr <= 0xBFFF && even:
		m.writeMirroring(val)
	case addr >= 0xA000 && addr <= 0xBFFF && odd:
		m.writePrgRamProtect(val)
	case addr >= 0xC000 && addr <= 0xDFFF && even:
		m.writeIrqLatch(val)
	case addr >= 0xC000 && addr <= 0xDFFF && odd:
		m.writeIrqReload(val)
	case addr >= 0xE000 && addr <= 0xFFFF && even:
		m.writeIrqDisable(val)
	case addr >= 0xE000 && addr <= 0xFFFF && odd:
		m.writeIrqEnable(val)
	}
	m.updateAllBanks()
}

func (m *MapperMMC3) updateAllBanks() {
	m.updateChrBanks()
	m.updatePgrBanks()
}

// Bank select ($8000-$9FFE, even)
//
// 7  bit  0
// ---- ----
// CPMx xRRR
// |||   |||
// |||   +++- Specify which bank register to update on next write to Bank Data register
// |||          000: R0: Select 2 KB CHR bank at PPU $0000-$07FF (or $1000-$17FF)
// |||          001: R1: Select 2 KB CHR bank at PPU $0800-$0FFF (or $1800-$1FFF)
// |||          010: R2: Select 1 KB CHR bank at PPU $1000-$13FF (or $0000-$03FF)
// |||          011: R3: Select 1 KB CHR bank at PPU $1400-$17FF (or $0400-$07FF)
// |||          100: R4: Select 1 KB CHR bank at PPU $1800-$1BFF (or $0800-$0BFF)
// |||          101: R5: Select 1 KB CHR bank at PPU $1C00-$1FFF (or $0C00-$0FFF)
// |||          110: R6: Select 8 KB PRG ROM bank at $8000-$9FFF (or $C000-$DFFF)
// |||          111: R7: Select 8 KB PRG ROM bank at $A000-$BFFF
// ||+------- Nothing on the MMC3, see MMC6
// |+-------- PRG ROM bank mode (0: $8000-$9FFF swappable,
// |                                $C000-$DFFF fixed to second-last bank;
// |                             1: $C000-$DFFF swappable,
// |                                $8000-$9FFF fixed to second-last bank)
// +--------- CHR A12 inversion (0: two 2 KB banks at $0000-$0FFF,
//                                  four 1 KB banks at $1000-$1FFF;
//                               1: two 2 KB banks at $1000-$1FFF,
//                                  four 1 KB banks at $0000-$0FFF)
func (m *MapperMMC3) writeBankSelect(val uint8) {
	m.bankMode = val
}

// Bank data ($8001-$9FFF, odd)
//
// 7  bit  0
// ---- ----
// DDDD DDDD
// |||| ||||
// ++++-++++- New bank value, based on last value written to Bank select register (mentioned above)
func (m *MapperMMC3) writeBankData(val uint8) {
	m.bankData = val
	m.registers[m.bankMode&7] = m.bankData
}

func (m *MapperMMC3) bank(register int) uint32 {
	return uint32(m.registers[register])
}
func (m *MapperMMC3) updateChrBanks() {
	for register, _ := range m.registers {
		m.updateChrBank(register)
	}
}
func (m *MapperMMC3) updateChrBank(register int) {
	chrInversion := (m.bankMode >> 7) == 1
	if !chrInversion {
		switch register {
		case 0, 1:
			m.chrBanks[0+register*2] = m.bank(register) * 0x400
			m.chrBanks[1+register*2] = m.bank(register)*0x400 + 0x400
		case 2, 3, 4, 5:
			m.chrBanks[register+2] = m.bank(register) * 0x400
		}
	} else {
		switch register {
		case 0, 1:
			m.chrBanks[4+register*2] = m.bank(register) * 0x400
			m.chrBanks[5+register*2] = m.bank(register)*0x400 + 0x400
		case 2, 3, 4, 5:
			m.chrBanks[register-2] = m.bank(register) * 0x400
		}
	}
}
func (m *MapperMMC3) updatePgrBanks() {
	for register, _ := range m.registers {
		m.updatePgrBank(register)
	}
	m.updatePgrBank(-2)
	m.updatePgrBank(-1)
}
func (m *MapperMMC3) updatePgrBank(register int) {
	prgInversion := (m.bankMode & 0x40) != 0
	if !prgInversion {
		switch register {
		case 6, 7:
			m.prgBanks[register-6] = m.bank(register) * 0x2000
		case -2:
			m.prgBanks[2] = uint32(m.cart.config.prgRomSize) - 0x4000
		case -1:
			m.prgBanks[3] = uint32(m.cart.config.prgRomSize) - 0x2000
		}
	} else {
		switch register {
		case 7:
			m.prgBanks[1] = m.bank(register) * 0x2000
		case 6:
			m.prgBanks[2] = m.bank(register) * 0x2000
		case -2:
			m.prgBanks[0] = uint32(m.cart.config.prgRomSize) - 0x4000
		case -1:
			m.prgBanks[3] = uint32(m.cart.config.prgRomSize) - 0x2000
		}
	}
}

// Mirroring ($A000-$BFFE, even)
//
// 7  bit  0
// ---- ----
// xxxx xxxM
//         |
//         +- Select nametable mirroring (0: vertical; 1: horizontal)
func (m *MapperMMC3) writeMirroring(val uint8) {
	m.mirror = val & 0x3
	switch m.mirror {
	case 0:
		m.cart.SetMirroring(common.VerticalMirroring)
	case 1:
		m.cart.SetMirroring(common.HorizontalMirroring)
	}
}

// PRG RAM protect ($A001-$BFFF, odd)
//
// 7  bit  0
// ---- ----
// RWXX xxxx
// ||||
// ||++------ Nothing on the MMC3, see MMC6
// |+-------- Write protection (0: allow writes; 1: deny writes)
// +--------- PRG RAM chip enable (0: disable; 1: enable)
func (m *MapperMMC3) writePrgRamProtect(val uint8) {
	m.prgRamProtect = val
}

// IRQ latch ($C000-$DFFE, even)
//
//7  bit  0
//---- ----
//DDDD DDDD
//|||| ||||
//++++-++++- IRQ latch value
func (m *MapperMMC3) writeIrqLatch(val uint8) {
	m.irqLatch = val
}

// IRQ reload ($C001-$DFFF, odd)
// Writing any value to this register reloads the MMC3 IRQ counter at
// the NEXT rising edge of the PPU address, presumably at
// PPU cycle 260 of the current scanline.
func (m *MapperMMC3) writeIrqReload(val uint8) {
	m.irqReload = val
}

// IRQ disable ($E000-$FFFE, even)
// Writing any value to this register will disable MMC3 interrupts AND
// acknowledge any pending interrupts.
func (m *MapperMMC3) writeIrqDisable(val uint8) {
	m.irqDisable = true
}

// IRQ enable ($E001-$FFFF, odd)
// Writing any value to this register will enable MMC3 interrupts.
func (m *MapperMMC3) writeIrqEnable(val uint8) {
	m.irqDisable = false
}

// CPU $6000-$7FFF: 8 KB PRG RAM bank (optional)
// CPU $8000-$9FFF (or $C000-$DFFF): 8 KB switchable PRG ROM bank
// CPU $A000-$BFFF: 8 KB switchable PRG ROM bank
// CPU $C000-$DFFF (or $8000-$9FFF): 8 KB PRG ROM bank, fixed to the second-last bank
// CPU $E000-$FFFF: 8 KB PRG ROM bank, fixed to the last bank
// PPU $0000-$07FF (or $1000-$17FF): 2 KB switchable CHR bank
// PPU $0800-$0FFF (or $1800-$1FFF): 2 KB switchable CHR bank
// PPU $1000-$13FF (or $0000-$03FF): 1 KB switchable CHR bank
// PPU $1400-$17FF (or $0400-$07FF): 1 KB switchable CHR bank
// PPU $1800-$1BFF (or $0800-$0BFF): 1 KB switchable CHR bank
// PPU $1C00-$1FFF (or $0C00-$0FFF): 1 KB switchable CHR bank
func (m *MapperMMC3) Read8(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		bank := addr / 0x400
		offset := uint32(addr) % 0x400
		return m.cart.chr.Read8w(m.chrBanks[bank] + offset)

	case addr >= 0x6000 && addr < 0x8000:
		return m.cart.prgRam.Read8(addr - 0x6000)

	case addr >= 0x8000:
		bank := (addr - 0x8000) / 0x2000
		offset := uint32(addr-0x8000) % 0x2000
		r := m.cart.prgRom.Read8w(m.prgBanks[bank] + offset)
		return r

	default:
		panic(fmt.Sprintf("read not implemented for 0x%04x!", addr))
	}
}
func (m *MapperMMC3) Write8(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		bank := addr / 0x400
		offset := uint32(addr) % 0x400
		m.cart.chr.Write8w(m.chrBanks[bank]+offset, val)

	case addr >= 0x6000 && addr < 0x8000:
		m.cart.prgRam.Write8(addr-0x6000, val)

	case addr >= 0x8000:
		m.writeInner(addr, val)
	default:
		panic(fmt.Sprintf("write not implemented for 0x%04x!", addr))
	}
}

func (m *MapperMMC3) Serialise(s common.Serialiser) error {
	return s.Serialise(
		m.bankMode, m.bankData, m.prgRamProtect, m.irqLatch,
		m.mirror, m.registers, m.prgBanks, m.chrBanks,
	)
}
func (m *MapperMMC3) DeSerialise(s common.Serialiser) error {
	return s.DeSerialise(
		&m.bankMode, &m.bankData, &m.prgRamProtect, &m.irqLatch,
		&m.mirror, &m.registers, &m.prgBanks, &m.chrBanks,
	)
}
