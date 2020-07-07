package nesInternal

import (
	"log"
)

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

func (m *cpuMapper) Read8(addr uint16) uint8 {
	switch {
	case addr < 0x2000:
		return m.nes.ram.Read8(addr % 2048)

	case addr < 0x4000:
		return m.nes.ppu.Read8(addr)

	case addr == 0x4015:
		return m.nes.apu.Read8(addr)
	case addr < 0x4016:
		// read from APU and I-O
		log.Panicf("read to address 0x%04x not implemented", addr)
	case addr < 0x4018:
		// Controller
		return m.nes.ctrl.Read8(addr)
	case addr < 0x4020:
		log.Panicf("read to address 0x%04x not implemented", addr)
	case addr < 0x6000:
		// todo: not sure what these are
		log.Panicf("read to address 0x%04x not implemented", addr)
	default:
		return m.nes.cart.Mapper.Read8(addr)
	}
	return 0
}

func (m *cpuMapper) Write8(addr uint16, val uint8) {
	switch {
	case addr < 0x2000:
		m.nes.ram.Write8(addr%2048, val)

	case addr < 0x4000:
		m.nes.ppu.Write8(addr, val)

	case addr < 0x4014, addr == 0x4015, addr == 0x4017:
		m.nes.apu.Write8(addr, val)

	case addr == 0x4014:
		m.nes.dma.Write8(addr, val)

	case addr < 0x4018:
		m.nes.ctrl.Write8(addr, val)

	case addr == 0x4025:
		// FDS not implemented

	case addr < 0x6000:
		// todo: not sure what these are
		log.Printf("write to address 0x%04x not implemented", addr)
	default:
		m.nes.cart.Mapper.Write8(addr, val)
	}
}

// DMA
// entity that handles writes to OAMDMA register
// it reads from the cpu and copies to the ppu OAMDATA register

type dmaMapper struct {
	*nes
}

func (m *dmaMapper) Read8(addr uint16) uint8 {
	// read from the cpu
	return m.nes.cpu.Read8(addr)
}

func (m *dmaMapper) Write8(addr uint16, val uint8) {
	// and copy to the ppu
	m.nes.ppu.Write8(addr, val)
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

func (m *ppuMapper) Read8(addr uint16) uint8 {
	switch {
	// PPU VRAM or controlled via the Cartridge Mapper
	case addr < 0x2000:
		return m.nes.cart.Mapper.Read8(addr)
	// normally mapped to the internal vRAM but it can be remapped!
	case addr < 0x3000:
		return m.nes.cart.Tables.Read8(addr)
	case addr < 0x3F00:
		return m.nes.cart.Tables.Read8(addr - 0x1000)

	// internal palette control - not configurable
	case addr < 0x4000:
		return m.nes.ppu.Palette.Read8(addr % 32)
	}
	return 0
}

func (m *ppuMapper) Write8(addr uint16, val uint8) {
	switch {
	// PPU VRAM or controlled via the Cartridge Mapper
	case addr < 0x2000:
		m.nes.cart.Mapper.Write8(addr, val)
	case addr < 0x3000:
		m.nes.cart.Tables.Write8(addr, val)
	case addr < 0x3F00:
		m.nes.cart.Tables.Write8(addr-0x1000, val)

	// internal palette control
	case addr < 0x4000:
		m.nes.ppu.Palette.Write8(addr%32, val)
	}
}

// APU
// Could also be used to keep track of time?
type apuMapper struct {
	*nes
}

func (a *apuMapper) Read8(addr uint16) uint8 {
	return a.cpu.Read8(addr)
}
func (a *apuMapper) Write8(uint16, uint8) {}
