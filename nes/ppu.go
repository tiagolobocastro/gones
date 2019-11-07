package gones

import (
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

	clock    int
	cycle    int
	scanLine int
	frames   int
	verbose  bool

	// cpu mapper registers
	regs [8]register

	// internal registers: http://wiki.nesdev.com/w/index.php/PPU_scrolling
	vRAM    register16 // Current VRAM address (15 bits)
	tRAM    register16 // Temporary VRAM address (15 bits); can also be thought of as the address of the top left onscreen tile.
	xFine   register   // Fine X scroll (3 bits)
	wToggle register   // First or second write toggle (1 bit)

	// background
	nametableEntry uint8
	attributeEntry uint8
	lowOrderByte   uint8
	highOrderByte  uint8

	// sprites
	rOAM ram
	// primary OAM
	pOAM [8]OamSprite
	// secondary OAM
	// In addition to the primary OAM memory, the PPU contains 32 bytes (enough for 8 sprites) of secondary OAM memory
	// that is not directly accessible by the program. During each visible scanline this secondary OAM is first cleared,
	// and then a linear search of the entire primary OAM is carried out to find sprites that are within y range for the
	// next scanline (the sprite evaluation phase). The OAM data for each sprite found to be within range is copied into
	// the secondary OAM, which is then used to initialize eight internal sprite output units.
	sOAM [8]OamSprite

	palette ppuPalette

	frameBuffer []color.RGBA

	buffered   bool
	backBuffer []color.RGBA

	interrupts iInterrupt
}

func (p *Ppu) init(busInt busInt, verbose bool, interrupts iInterrupt, frameBuffer []color.RGBA) {
	p.verbose = verbose
	p.busInt = busInt
	p.interrupts = interrupts
	p.clock = 0
	p.cycle = 0
	p.scanLine = 0
	p.frames = 0
	p.frameBuffer = frameBuffer

	p.buffered = false
	if p.buffered {
		p.backBuffer = make([]color.RGBA, 256*240)
	}

	p.vRAM.init("v", 0)
	p.tRAM.init("t", 0)
	p.xFine.init("x", 0)
	p.wToggle.init("w", 0)
	p.rOAM.initNfill(256, 0xfe)
	p.palette.init()

	p.initRegisters()
	p.clearSecOAM()
	p.clearPrimOAM()
}

func (p *Ppu) reset() {
	p.init(p.busInt, p.verbose, p.interrupts, p.frameBuffer)
}

// interrupt
// only look at the CPU NMI for now
// need to implement the interrupt delay as well since the cpu and ppu and not on the same clock
func (p *Ppu) raise(flag uint8) {
	if (flag & cpuIntNMI) != 0 {

		if p.buffered {
			p.backBuffer, p.frameBuffer = p.frameBuffer, p.backBuffer
		}

		p.frames++

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

// start easy with a dummy imp
func (p *Ppu) fetchNameTableEntry() {

	p.nametableEntry = p.busInt.read8(0x2000 + uint16(p.scanLine/8)*32 + uint16(p.cycle/8))
}

func (p *Ppu) fetchAttributeTableEntry() {
	p.attributeEntry = p.busInt.read8(0x23C0 + uint16(p.scanLine/32)*8 + uint16(p.cycle/32))
}

func (p *Ppu) fetchLowOrderByte() {
	p.lowOrderByte = p.busInt.read8(0x1000 + uint16(p.nametableEntry)*16 + uint16(p.scanLine%8))
}

func (p *Ppu) fetchHighOrderByte() {
	p.highOrderByte = p.busInt.read8(0x1000 + uint16(p.nametableEntry)*16 + uint16(p.scanLine%8) + 8)
}

func (p *Ppu) combineBS() {
}

func (p *Ppu) exec() {

	if p.scanLine < 240 {
		switch p.cycle {
		// the ppu "works" these every cycle and it might more efficient for us to do the same
		// but now for simplicity let's bundle each task
		case 1:
			p.clearSecOAM()
			if p.scanLine == -1 {
				p.regs[PPUSTATUS].clr(statusSpriteOverflow | statusSprite0Hit)
			}
		case 257:
			p.evalSprites()
		case 321:
			p.loadSprites()
		}
	}

	var c color.RGBA

	// background
	if p.scanLine > -1 && p.scanLine < 240 && p.cycle < 256 {
		/*
			switch p.cycle%8 {
			case 1:
				p.fetchNameTableEntry()
			case 3:
				p.fetchAttributeTableEntry()
			case 5:
				p.fetchLowOrderByte()
			case 7:
				p.fetchHighOrderByte()
			case 0:
				p.combineBS()
			}
		*/

		x := uint8(p.cycle)
		y := uint8(p.scanLine)

		p.fetchNameTableEntry()
		p.fetchAttributeTableEntry()
		p.fetchLowOrderByte()
		p.fetchHighOrderByte()

		bit := uint8(8 - p.cycle%8 - 1)

		b0 := (p.lowOrderByte >> bit) & 1
		b1 := (p.highOrderByte >> bit) & 1
		b := uint16(b0 | (b1 << 1))

		palette := uint16(p.attributeEntry)
		i := (x/16)%2 | ((y/16)%2)<<1
		palette = (palette >> (2 * i)) & 3

		// 4 background + 4 sprite palettes
		index := p.busInt.read8(0x3F00 + (palette)*4 + b)
		c = p.palette.nesPalette[index]
		//p.drawPixel(x, y, c)
	}

	if p.scanLine > -1 && p.scanLine < 240 && p.cycle < 256 {
		x := uint8(p.cycle)
		y := uint8(p.scanLine)

		for i := range p.pOAM {
			s := &p.pOAM[i]

			xi := uint(x) - uint(s.xPos)
			yi := uint(y) - uint(s.yPos)

			if yi < 8 && xi < 8 {

				bit := 8 - xi - 1

				b0 := (s.lsbIndex >> bit) & 1
				b1 := (s.msbIndex >> bit) & 1
				b := uint16(b0 | (b1 << 1))

				palette := uint16(s.attributes & 0x3)

				// 4 background + 4 sprite palettes
				index := p.busInt.read8(0x3F00 + (palette+4)*4 + b)
				if b != 0 {
					c = p.palette.nesPalette[index]
				}
				break
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

		if p.scanLine > 260 {
			p.scanLine = -1
			// may already be cleared as reading from PPSTATUS will do so
			p.clear(cpuIntNMI)
		}
	}
}

func (p *Ppu) drawPixel(x uint8, y uint8, c color.RGBA) {
	if !p.buffered {
		p.frameBuffer[(240-1-uint16(y))*256+uint16(x)] = c
	} else {
		p.backBuffer[(240-1-uint16(y))*256+uint16(x)] = c
	}
}

func (p *Ppu) loadSprites() {
	_, spriteSizeY := p.getSpriteSize()
	patternAddr := p.getSpritePattern()
	for i := range p.sOAM {

		if !p.sOAM[i].set {
			continue
		}

		p.pOAM[i] = p.sOAM[i]
		s := &p.pOAM[i]

		addr := uint16(0)
		if spriteSizeY == 16 {
			// taken from HydraNes, have not verified this
			addr = ((uint16(s.tIndex) & 1) * p.getSpritePattern()) + ((uint16(s.tIndex) & (1 ^ 0xFFFF)) * 16)
		} else {
			addr = patternAddr + uint16(s.tIndex)*16
		}

		// calculate line inside sprite for the next scanLine
		lSpY := (p.scanLine + 1 - int(s.yPos)) % int(spriteSizeY)

		// vertical flip
		if (s.attributes & 0x80) != 0 {
			lSpY ^= int(spriteSizeY) - 1
		}

		addr += uint16(lSpY) + uint16(lSpY&8)

		s.lsbIndex = p.busInt.read8(addr)
		s.msbIndex = p.busInt.read8(addr + 8)

		// horizontal flip
		if (s.attributes & 0x40) != 0 {
			s.lsbIndex = reverseByte(s.lsbIndex)
			s.msbIndex = reverseByte(s.msbIndex)
		}
	}
}

func reverseByte(b uint8) uint8 {
	return ((b & 0x01) << 7) | ((b & 0x02) << 5) |
		((b & 0x04) << 3) | ((b & 0x08) << 1) |
		((b & 0x10) >> 1) | ((b & 0x20) >> 3) |
		((b & 0x40) >> 5) | ((b & 0x80) >> 7)
}

func (p *Ppu) evalSprites() {
	spriteCount := 0
	evalScan := p.scanLine + 1
	_, yLen := p.getSpriteSize()
	for i := uint16(0); i < 64; i++ {

		// 0 yPos, 1 index, 2 attr, 3 xPos => i*4
		yPos := p.rOAM.read8(i * 4)
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

func (p *Ppu) clearPrimOAM() {
	for i := range p.pOAM {
		// set back defaults
		p.pOAM[i] = OamSprite{
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

func (p *Ppu) clearSecOAM() {
	for i := range p.sOAM {
		// set back defaults
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

func (p *Ppu) tick() {
	p.clock++
	p.exec()
}

func (p *Ppu) ticks(nTicks int) {

	for i := 0; i < nTicks; i++ {
		p.tick()
	}
}

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
