package gones

type Ppu struct {
	busInt

	clk int64

	palette ppuPalette

	verbose      bool
	disableBreak bool
}

func (p *Ppu) init(busInt busInt, verbose bool) {
	p.verbose = verbose

	p.busInt = busInt
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
	case addr < 0x3F20:
		m.nes.ppu.palette.write8(addr%32, val)
	case addr < 0x4000:
		m.nes.ppu.palette.write8(addr%32, val)
	}
}

// cpu can read from the ppu through the control registers

// BusInt
func (p *Ppu) read8(addr uint16) uint8 {
	return 0
}
func (p *Ppu) write8(addr uint16, val uint8) {

}
