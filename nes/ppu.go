package gones

import (
	"golang.org/x/image/colornames"
	"image/color"
)

// http://wiki.nesdev.com/w/index.php/PPU_OAM
type OamSprite struct {
	// Y position of top of sprite
	yPos uint8
	// Tile index number
	tIndex uint8
	// Sprite Attributes
	attributes uint8
	// X position of left side of sprite
	xPos uint8

	// data
	msbIndex uint8
	lsbIndex uint8

	// clr
	set bool
}

type Ppu struct {
	busInt

	cycle    int
	scanLine int
	verbose  bool

	// cpu mapper registers
	regs [8]register

	// internal registers: http://wiki.nesdev.com/w/index.php/PPU_scrolling
	vRAM    register16 // Current VRAM address (15 bits)
	tRAM    register16 // Temporary VRAM address (15 bits); can also be thought of as the address of the top left onscreen tile.
	xFine   register   // Fine X scroll (3 bits)
	wToggle register   // First or second write toggle (1 bit)

	rOAM ram
	// primary OAM
	pOAM [64]OamSprite
	// secondary OAM
	// In addition to the primary OAM memory, the PPU contains 32 bytes (enough for 8 sprites) of secondary OAM memory
	// that is not directly accessible by the program. During each visible scanline this secondary OAM is first cleared,
	// and then a linear search of the entire primary OAM is carried out to find sprites that are within y range for the
	// next scanline (the sprite evaluation phase). The OAM data for each sprite found to be within range is copied into
	// the secondary OAM, which is then used to initialize eight internal sprite output units.
	sOAM [8]OamSprite

	palette ppuPalette

	frameBuffer []color.RGBA

	interrupts iInterrupt
}

func (p *Ppu) init(busInt busInt, verbose bool, interrupts iInterrupt, frameBuffer []color.RGBA) {
	p.verbose = verbose
	p.busInt = busInt
	p.interrupts = interrupts
	p.cycle = 0
	p.scanLine = 0
	p.frameBuffer = frameBuffer

	p.vRAM.init("v", 0)
	p.tRAM.init("t", 0)
	p.xFine.init("x", 0)
	p.wToggle.init("w", 0)
	p.rOAM.initNfill(256, 0xfe)

	p.initRegisters()
}

func (p *Ppu) reset() {
	p.init(p.busInt, p.verbose, p.interrupts, p.frameBuffer)
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
	case addr < 0x2000:
		return m.nes.cart.mapper.read8(addr % 2048)
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

// interrupt
// only look at the CPU NMI for now
// need to implement the interrupt delay as well since the cpu and ppu and not on the same clock
func (p *Ppu) raise(flag uint8) {
	if (flag & cpuIntNMI) != 0 {
		p.regs[PPUSTATUS].val |= 0x80

		if p.getNMIVertical() == 1 {
			p.interrupts.raise(flag & cpuIntNMI)
		}
	}
}
func (p *Ppu) clear(flag uint8) {
	if (flag & cpuIntNMI) != 0 {
		p.regs[PPUSTATUS].val &= 0x7F
		p.interrupts.clear(flag & cpuIntNMI)
	}
}

func (p *Ppu) exec() {

	if p.scanLine < 240 || p.scanLine == 261 {
		switch p.cycle {
		case 1:
			p.clearSecOAM()
			if p.scanLine == 261 {
				p.regs[PPUSTATUS].clr(statusSpriteOverflow | statusSprite0Hit)
			}
		case 257:
			p.evalSprites()
		case 321:
			p.loadSprites()
		}
	}

	if p.scanLine < 240 && p.cycle < 256 {
		x := uint8(p.cycle)
		y := uint8(p.scanLine)
		c := colornames.Aliceblue

		palette := [4]color.RGBA{
			{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, // B - W
			{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}, // 1 - R
			{R: 0x00, G: 0x00, B: 0xFF, A: 0xFF}, // 2 - B
			{R: 0x00, G: 0xF0, B: 0xF0, A: 0xFF}, // 3 - LB
		}

		for _, s := range p.pOAM {
			if s.xPos <= x && x <= (s.xPos+7) {
				if s.yPos <= y && y <= (s.yPos+7) && s.tIndex < 255 && s.set {

					xi := x - s.xPos

					b0 := (s.lsbIndex >> (8 - xi - 1)) & 1
					b1 := (s.msbIndex >> (8 - xi - 1)) & 1
					b := b0 | (b1 << 1)

					c = palette[b]
					break
				}
			}
		}

		p.drawPixel(x, y, c)
	}

	p.cycle += 1

	if p.cycle > 340 {
		p.scanLine += 1
		p.cycle = 0

		if p.scanLine == 241 {
			p.raise(cpuIntNMI)
		}

		if p.scanLine > 261 {
			p.scanLine = 0
			// may already be cleared as reading from PPSTATUS will do so
			p.clear(cpuIntNMI)
		}
	}
}

func (p *Ppu) drawPixel(x uint8, y uint8, c color.RGBA) {
	//p.screen.drawPixel(x, y, c)
	if y > 32 {
		return
	}
	p.frameBuffer[(240-1-uint16(y))*256+uint16(x)] = c
}

func (p *Ppu) loadSprites() {
	for i := range p.sOAM {
		if !p.sOAM[i].set {
			continue
		}

		p.pOAM[i] = p.sOAM[i]
		_, spriteSizeY := p.getSpriteSize()

		addr := uint16(0)
		if spriteSizeY == 16 {
			// taken from HydraNes, have not verified this
			addr = ((uint16(p.pOAM[i].tIndex) & 1) * p.getSpritePattern()) + ((uint16(p.pOAM[i].tIndex) & (1 ^ 0xFFFF)) * 16)
		} else {
			addr = p.getSpritePattern() + uint16(p.pOAM[i].tIndex)*16
		}

		// calculate line inside sprite for the next scanLine
		lSpY := (p.scanLine + 1 - int(p.pOAM[i].yPos)) % int(spriteSizeY)

		// vertical flip
		if (p.pOAM[i].attributes & 0x80) != 0 {
			lSpY ^= int(spriteSizeY) - 1
		}

		addr += uint16(lSpY) + uint16(lSpY&8)

		p.pOAM[i].lsbIndex = p.busInt.read8(addr)
		p.pOAM[i].msbIndex = p.busInt.read8(addr + 8)

		// horizontal flip
		if (p.pOAM[i].attributes & 0x40) != 0 {
			p.pOAM[i].lsbIndex = reverseByte(p.pOAM[i].lsbIndex)
			p.pOAM[i].msbIndex = reverseByte(p.pOAM[i].msbIndex)
		}
	}
}

func reverseByte(b uint8) uint8 {
	return ((b & 0x1) << 7) | ((b & 0x2) << 5) |
		((b & 0x4) << 3) | ((b & 0x8) << 1) |
		((b & 0x10) >> 1) | ((b & 0x20) >> 3) |
		((b & 0x40) >> 5) | ((b & 0x80) >> 7)
}

func (p *Ppu) evalSprites() {
	spriteCount := 0
	evalScan := p.scanLine + 1
	for i := uint16(0); i < 64; i++ {

		yPos := p.rOAM.read8(i*4 + 0)
		_, yLen := p.getSpriteSize()
		yPosEnd := yPos + yLen

		// if the scanLine intersects the sprite, it's a "hit"
		// copy sprite to the secondary OAM
		if yPosEnd > yPos && evalScan >= int(yPos) && evalScan <= int(yPosEnd) {
			p.sOAM[spriteCount].yPos = p.rOAM.read8(i*4 + 0)
			p.sOAM[spriteCount].tIndex = p.rOAM.read8(i*4 + 1)
			p.sOAM[spriteCount].attributes = p.rOAM.read8(i*4 + 2)
			p.sOAM[spriteCount].xPos = p.rOAM.read8(i*4 + 3)
			p.sOAM[spriteCount].set = true

			spriteCount += 1
			if spriteCount >= 8 {
				p.regs[PPUSTATUS].set(statusSpriteOverflow)
				break
			}
		}
	}
}

func (p *Ppu) clearSecOAM() {
	for i := range p.sOAM {
		// set back to defaults
		p.sOAM[i] = OamSprite{
			yPos:       0xFF,
			tIndex:     0xFF,
			attributes: 0xFF,
			xPos:       0xFF,
			lsbIndex:   0x00,
			msbIndex:   0x00,
			set:        false,
		}
	}
}

func (p *Ppu) ticks(nTicks int) {

	for i := 0; i < nTicks; i++ {
		p.exec()
	}
}

func (p *Ppu) tick() {

	// first do the work, and only then tick?

	// let's add a simple sprite display or something like that
	// so let's do the bare minimum for the ppu setup

	p.tick()
}

// cpu can read from the ppu through the control registers

// BusInt
func (p *Ppu) read8(addr uint16) uint8 {
	if addr < 0x4000 {
		// incomplete decoding means 0x2000-0x2007 are mirrored every 8 bytes
		addr = 0x2000 + addr%8
	}

	switch addr {
	// PPU Status (PPUSTATUS) - RDONLY
	case 0x2002:
		return p.regs[PPUSTATUS].read()
	// PPU OAM Data (OAMDATA)
	case 0x2004:
		return p.regs[OAMDATA].read()
	// PPU Data (PPUDATA)
	case 0x2007:
		return p.regs[PPUDATA].read()
	}

	return 0
}
func (p *Ppu) write8(addr uint16, val uint8) {
	if addr < 0x4000 {
		// incomplete decoding means 0x2000-0x2007 are mirrored every 8 bytes
		addr = 0x2000 + addr%8
	}

	switch addr {
	// PPU Control (PPUCTRL) - WRONLY
	case 0x2000:
		p.regs[PPUCTRL].write(val)
	// PPU Mask (PPUMASK) - WRONLY
	case 0x2001:
		p.regs[PPUMASK].write(val)
	// PPU OAM Data (OAMADDR) - WRONLY
	case 0x2003:
		p.regs[OAMADDR].write(val)
	// PPU OAM Data (OAMDATA)
	case 0x2004:
		p.regs[OAMDATA].write(val)
	// PPU Scrolling (PPUSCROLL) - WRONLY
	case 0x2005:
		p.regs[PPUSCROLL].write(val)
	// PPU Address (PPUADDR) - WRONLY
	case 0x2006:
		p.regs[PPUADDR].write(val)
	// PPU Data (PPUDATA)
	case 0x2007:
		p.regs[PPUDATA].write(val)
	// PPU OAM DMA (OAMDMA) - WRONLY
	case 0x4014:
		// handled by the dma engine
		panic("OAMDMA should have gone to the dma engine!")
	}
}
