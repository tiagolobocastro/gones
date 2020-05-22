package gones

import "fmt"

const (
	mapperNROM = iota
	mapperMMC1
	mapperUnROM
)

type Mapper interface {
	busInt
	Init()
}

type MapperNROM struct {
	cart *Cartridge
}

func (m *MapperNROM) Init() {}

//CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
//CPU $8000-$BFFF: First 16 KB of ROM.
//CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
func (m *MapperNROM) read8(addr uint16) uint8 {
	switch {
	// PPU - normally mapped by the cartridge to a CHR-ROM or CHR-RAM,
	// often with a bank switching mechanism.
	case addr < 0x2000:
		return m.cart.chr.read8(addr)
	case addr < 0x8000:
		return m.cart.prgRam.read8(addr)
	default:
		return m.cart.prgRom.read8(uint16(int(addr) % m.cart.prgRom.size()))
	}
}
func (m *MapperNROM) write8(addr uint16, val uint8) {
	switch {
	case addr > 0x6000, addr < 0x8000:
		m.cart.prgRam.write8(addr%0x6000, val)
	default:
		panic(fmt.Sprintf("write not implemented for %v!", addr))
	}
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
		//panic("address range not implemented!")
	case addr < 0x4018:
		// Controller
		return m.nes.ctrl.read8(addr)
	case addr < 0x4020:
		// APU
		panic(fmt.Errorf("address range not implemented, addr: 0x%04x", addr))

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
		//panic("address range not implemented!")
	case addr < 0x4018:
		// Controller
		m.nes.ctrl.write8(addr, val)
	case addr < 0x4020:
		// APU
		//panic("address range not implemented!")

	default:
		m.nes.cart.mapper.write8(addr, val)
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
//
// The mappings above are the fixed addresses from which the PPU uses to fetch data during rendering.
// The actual device that the PPU fetches data from, however, may be configured by the cartridge.
//
//  $0000-1FFF is normally mapped by the cartridge to a CHR-ROM or CHR-RAM, often with a bank switching mechanism.
//
//  $2000-2FFF is normally mapped to the 2kB NES internal VRAM, providing 2 nametables with a mirroring
//       configuration controlled by the cartridge, but it can be partly or fully remapped to RAM on the cartridge,
//       allowing up to 4 simultaneous nametables.
//
//  $3000-3EFF is usually a mirror of the 2kB region from $2000-2EFF. The PPU does not render from this address range,
//       so this space has negligible utility.
//
//  $3F00-3FFF is not configurable, always mapped to the internal palette control.
type ppuMapper struct {
	*nes
}

func (m *ppuMapper) read8(addr uint16) uint8 {
	switch {
	// PPU VRAM or controlled via the Cartridge Mapper
	case addr < 0x2000:
		return m.nes.cart.mapper.read8(addr)
	// normally mapped to the internal vRAM but it can be remapped!
	case addr < 0x3000:
		return m.nes.cart.tables.read8(addr)
	case addr < 0x3F00:
		return m.nes.cart.tables.read8(addr - 0x1000)

	// internal palette control - not configurable
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
	case addr < 0x2000:
		m.nes.cart.mapper.write8(addr, val)
	case addr < 0x3000:
		m.nes.cart.tables.write8(addr, val)
	case addr < 0x3F00:
		m.nes.cart.tables.write8(addr-0x1000, val)

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
	case addr >= 0xE000 && addr <= 0xFFFF:
		m.writePRGBank(val)
	}
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
		m.cart.tables.mirroring = SingleScreenMirroring
	case 1:
		m.cart.tables.mirroring = SingleScreenMirroring
	case 2:
		m.cart.tables.mirroring = VerticalMirroring
	case 3:
		m.cart.tables.mirroring = HorizontalMirroring
	}
	m.prgBankMode = (val >> 2) & 0x3
	m.chrBankMode = val >> 4
}

// CHR bank 0 (internal, $A000-$BFFF)
//
// 4bit0
// -----
// CCCCC
// |||||
// +++++- Select 4 KB or 8 KB CHR bank at PPU $0000 (low bit ignored in 8 KB mode)
func (m *MapperMMC1) writeCHRBank0(val uint8) {
	switch m.chrBankMode {
	case 0:
		// 8 KB
		bank := (uint16(val&0x1e) >> 1) * 0x2000
		m.chrBanks[0] = bank
		m.chrBanks[1] = bank + 0x1000
	case 1:
		// 4 KB
		bank := uint16(val&0x1f) * 0x1000
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
	switch m.chrBankMode {
	case 0:
		// 8 KB
		// noop
	case 1:
		// 4 KB
		bank := uint16(val&0x1f) * 0x1000
		m.chrBanks[1] = bank + 0x1000
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
	switch m.prgBankMode {
	case 0, 1:
		// 32KB mode
		bankV := uint32(val&0xe) >> 1
		bank := 0x8000 * bankV

		m.prgBanks[0] = bank
		m.prgBanks[1] = bank + 0x4000

	case 2:
		//  2: fix first bank at $8000 and switch 16 KB bank at $C000;
		m.prgBanks[0] = 0x0000
		m.prgBanks[1] = 0x4000 * uint32(val)
	case 3:
		// 3: fix last bank at $C000 and switch 16 KB bank at $8000)
		m.prgBanks[0] = 0x4000 * uint32(val)
		m.prgBanks[1] = (0xe >> 1) * 0x4000
	}
}

func (m *MapperMMC1) Init() {
}

//CPU $6000-$7FFF: Family Basic only: PRG RAM, mirrored as necessary to fill entire 8 KiB window, write protectable with an external switch
//CPU $8000-$BFFF: First 16 KB of ROM.
//CPU $C000-$FFFF: Last 16 KB of ROM (NROM-256) or mirror of $8000-$BFFF (NROM-128).
func (m *MapperMMC1) read8(addr uint16) uint8 {
	switch {
	// PPU - normally mapped by the cartridge to a CHR-ROM or CHR-RAM,
	// often with a bank switching mechanism.
	case addr < 0x1000:
		return m.cart.chr.read8(addr + m.chrBanks[0])
	case addr < 0x2000:
		return m.cart.chr.read8(addr - 0x1000 + m.chrBanks[1])
	case addr > 0x6000 && addr < 0x8000:
		return m.cart.prgRam.read8(addr % 0x6000)
	case addr >= 0x8000 && addr < 0xC000:
		offset := uint32(addr - 0x8000)
		return m.cart.prgRom.read8w(m.prgBanks[0] + offset)
	case addr > 0xC000:
		offset := uint32(addr - 0xC000)
		return m.cart.prgRom.read8w(m.prgBanks[1] + offset)
	default:
		panic(fmt.Sprintf("write not implemented for 0x%04x!", addr))
	}
}
func (m *MapperMMC1) write8(addr uint16, val uint8) {
	switch {
	case addr < 0x1000:
		m.cart.chr.write8(addr+m.chrBanks[0], val)
	case addr < 0x2000:
		m.cart.chr.write8(addr-0x1000+m.chrBanks[1], val)
	case addr >= 0x6000 && addr < 0x8000:
		m.cart.prgRam.write8(addr%0x6000, val)
	case addr >= 0x8000:
		m.writeLoad(addr, val)
	default:
		panic(fmt.Sprintf("write not implemented for 0x%04x!", addr))
	}
}
