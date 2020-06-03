package ppu

import (
	"github.com/tiagolobocastro/gones/nes/cpu"
	"image/color"

	"github.com/tiagolobocastro/gones/nes/common"
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

	// sprite id
	id uint8

	// data
	msbIndex uint8
	lsbIndex uint8
}

type Ppu struct {
	common.BusInt

	clock    int
	cycle    int
	scanLine int
	frames   int
	verbose  bool

	// cpu mapper registers
	regs [8]common.Register

	// internal registers: http://wiki.nesdev.com/w/index.php/PPU_scrolling
	vRAM    loopyRegister   // Current VRAM address (15 bits)
	tRAM    loopyRegister   // Temporary VRAM address (15 bits); can also be thought of as the address of the top left onscreen tile.
	xFine   common.Register // Fine X scroll (3 bits)
	wToggle common.Register // First or second write toggle (1 bit)

	// background
	nametableEntry uint8
	attributeEntry uint8
	lowOrderByte   uint8
	highOrderByte  uint8
	tileData       uint64
	rowShifter     uint64

	nameTable uint8
	xScroll   uint8

	vRAMBuffer uint8

	// sprites
	rOAM common.Ram
	// primary OAM
	pOAM [8]OamSprite
	// secondary OAM
	// In addition to the primary OAM memory, the PPU contains 32 bytes (enough for 8 sprites) of secondary OAM memory
	// that is not directly accessible by the program. During each visible scanline this secondary OAM is first cleared,
	// and then a linear search of the entire primary OAM is carried out to find sprites that are within Y range for the
	// next scanline (the sprite evaluation phase). The OAM data for each sprite found to be within range is copied into
	// the secondary OAM, which is then used to initialize eight internal sprite output units.
	sOAM [8]OamSprite

	// move this into a struct maybe
	bgIndex    uint8
	bgPalette  uint8
	fgIndex    uint8
	fgPalette  uint8
	fgPriority bool

	Palette ppuPalette

	frameBuffer *common.Framebuffer
	buffered    bool

	interrupts common.IiInterrupt

	finalScroll uint8
}

func (p *Ppu) Init(busInt common.BusInt, verbose bool, interrupts common.IiInterrupt, framebuffer *common.Framebuffer) {
	p.verbose = verbose
	p.BusInt = busInt
	p.interrupts = interrupts
	p.clock = 0
	p.cycle = 0
	p.scanLine = -1
	p.frameBuffer = framebuffer
	p.buffered = true

	p.rOAM.InitNfill(256, 0xfe)
	p.Palette.init()

	p.initRegisters()
	p.clearOAM()
}

func (p *Ppu) Reset() {
	p.Init(p.BusInt, p.verbose, p.interrupts, p.frameBuffer)
}

// interrupt
// only look at the CPU NMI for now
// need to implement the interrupt delay as well since the cpu and ppu and not on the same clock
func (p *Ppu) raise(flag uint8) {
	if (flag & cpu.CpuIntNMI) != 0 {

		p.frameBuffer.Frames++

		if p.buffered {
			p.frameBuffer.FrameIndex ^= 1
		}

		select {
		case p.frameBuffer.FrameUpdated <- true:
			// todo: control "vsync" channel
			//default:
		}

		p.regs[PPUSTATUS].Val |= 0x80

		if p.getNMIVertical() == 1 {
			p.interrupts.Raise(flag & cpu.CpuIntNMI)
		}
	}
}
func (p *Ppu) clear(flag uint8) {
	if (flag & cpu.CpuIntNMI) != 0 {
		p.regs[PPUSTATUS].Val &= 0x7F
		p.interrupts.Clear(flag & cpu.CpuIntNMI)

		p.regs[PPUSTATUS].Clr(statusSpriteOverflow | statusSprite0Hit)
	}
}

func (p *Ppu) getNameTable() uint16 {
	nta := [2]uint16{}
	if p.nameTable == 0 {
		nta = [2]uint16{0x2000, 0x2400}
	} else {
		nta = [2]uint16{0x2400, 0x2000}
	}

	if (p.cycle + int(p.finalScroll)) > 255 {
		return nta[1]
	} else {
		return nta[0]
	}
}

func (p *Ppu) getAttributeNameTable() uint16 {
	nta := [2]uint16{}
	if p.nameTable == 0 {
		nta = [2]uint16{0x23C0, 0x27C0}
	} else {
		nta = [2]uint16{0x27C0, 0x23C0}
	}

	if (p.cycle + int(p.finalScroll)) > 255 {
		return nta[1]
	} else {
		return nta[0]
	}
}

// start easy with a dummy imp
func (p *Ppu) fetchNameTableEntry() {
	x := (p.cycle + int(p.finalScroll)) % 256
	addr := p.getNameTable() + uint16(p.scanLine/8)*32 + uint16(x/8)
	p.nametableEntry = p.BusInt.Read8(addr)
}

func (p *Ppu) fetchAttributeTableEntry() {
	x := (p.cycle + int(p.finalScroll)) % 256
	addr := p.getAttributeNameTable() + uint16(p.scanLine/32)*8 + uint16(x/32)
	p.attributeEntry = p.BusInt.Read8(addr)
}

func (p *Ppu) fetchLowOrderByte() {
	table := p.getBackgroundTable()
	addr := table + uint16(p.nametableEntry)*16 + uint16(p.scanLine%8)
	p.lowOrderByte = p.BusInt.Read8(addr)
}

func (p *Ppu) fetchHighOrderByte() {
	table := p.getBackgroundTable()
	addr := table + uint16(p.nametableEntry)*16 + uint16(p.scanLine%8)
	p.highOrderByte = p.BusInt.Read8(addr + 8)
}

func (p *Ppu) execOldPpu() {

	if p.scanLine < 240 {
		switch p.cycle {
		// the ppu "works" these every cycle and it might more efficient for us to do the same
		// but now for simplicity let's bundle each task
		case 1:
			p.clearSecOAM()
		case 257:
			p.evalSprites()
		case 321:
			p.loadSprites()
		}
	}

	// setup values required for the draw decision
	x := uint8(p.cycle)
	y := uint8(p.scanLine)
	p.bgIndex = 0
	p.bgPalette = 0
	p.fgIndex = 0
	p.fgPalette = 0
	p.fgPriority = false

	// http://wiki.nesdev.com/w/images/d/d1/Ntsc_timing.png
	visibleFrame := p.scanLine >= 0 && p.scanLine < 240
	preRenderLn := p.scanLine == -1
	renderFrame := visibleFrame || preRenderLn

	// cycle 0 is skipped for BG+odd => background and odd sprite frames?
	// cycle 337-340 are unused
	visibleCycle := p.cycle >= 0 && p.cycle <= 255

	// background
	if renderFrame && visibleCycle && p.showBackground() {

		if p.scanLine > 0 && p.scanLine%32 == 0 {
			p.nameTable = p.regs[PPUCTRL].Val & 3
		}

		p.fetchNameTableEntry()
		p.fetchAttributeTableEntry()
		p.fetchLowOrderByte()
		p.fetchHighOrderByte()

		xx := (p.cycle + int(p.finalScroll)) % 256
		bit := uint8(8 - xx%8 - 1)

		b0 := (p.lowOrderByte >> bit) & 1
		b1 := (p.highOrderByte >> bit) & 1
		p.bgIndex = b0 | (b1 << 1)

		palette := p.attributeEntry

		i := (uint8(xx)/16)%2 | ((y/16)%2)<<1
		p.bgPalette = (palette >> (2 * i)) & 3
	}

	if visibleFrame && visibleCycle && p.showSprites() {
		for i := range p.pOAM {
			if p.pOAM[i].id == 64 {
				continue
			}

			s := &p.pOAM[i]

			xi := uint(x) - uint(s.xPos)

			if xi < 8 {

				bit := 8 - xi - 1

				b0 := (s.lsbIndex >> bit) & 1
				b1 := (s.msbIndex >> bit) & 1
				p.fgIndex = b0 | (b1 << 1)
				p.fgPriority = (s.attributes>>5)&1 == 0
				p.fgPalette = s.attributes & 0x3

				// non transparent pixel found so "accept" this sprite
				if p.fgIndex != 0 {

					if s.id == 0 && p.bgIndex > 0 && x != 255 {
						p.regs[PPUSTATUS].Set(statusSprite0Hit)
					}

					break
				}
			}
		}
	}

	if visibleFrame && visibleCycle {

		// what gets drawn based on transparency (index==0) and priority
		if p.bgIndex == 0 && p.fgIndex == 0 {
			p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00)])
		} else if p.bgIndex > 0 && p.fgIndex == 0 {
			p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16(p.bgPalette*4+p.bgIndex))])
		} else if p.bgIndex == 0 && p.fgIndex > 0 {
			p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16((p.fgPalette+4)*4+p.fgIndex))])
		} else if p.bgIndex > 0 && p.fgIndex > 0 {
			if p.fgPriority {
				p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16((p.fgPalette+4)*4+p.fgIndex))])
			} else {
				p.drawPixel(x, y, p.Palette.nesPalette[p.BusInt.Read8(0x3F00+uint16(p.bgPalette*4+p.bgIndex))])
			}
		}
	}

	p.cycle += 1

	if p.cycle == 257 {
		p.finalScroll = p.xScroll
	}
	if p.cycle > 340 {

		p.scanLine += 1
		p.cycle = 0

		if p.scanLine > 260 {
			p.scanLine = -1
			// may already be cleared as reading from PPSTATUS will do so
			p.clear(cpu.CpuIntNMI)
		} else if p.scanLine == 241 {
			p.raise(cpu.CpuIntNMI)
		} else if p.scanLine == 242 {
			p.nameTable = p.regs[PPUCTRL].Val & 0x3
		}
	}
}

func (p *Ppu) drawPixel(x uint8, y uint8, c color.RGBA) {
	if p.buffered && p.frameBuffer.FrameIndex == 0 {
		p.frameBuffer.Buffer0[(240-1-uint16(y))*256+uint16(x)] = c
	} else {
		p.frameBuffer.Buffer1[(240-1-uint16(y))*256+uint16(x)] = c
	}
}

func (p *Ppu) loadSprites() {
	_, spriteSizeY := p.getSpriteSize()
	patternAddr := p.getSpritePattern()
	for i := range p.sOAM {

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
		// edit: seems like sprites are already arranged like so, meaning we can use the current?
		lSpY := (p.scanLine - int(s.yPos)) % int(spriteSizeY)

		// vertical flip
		if (s.attributes & 0x80) != 0 {
			lSpY ^= int(spriteSizeY) - 1
		}

		addr += uint16(lSpY) + uint16(lSpY&8)

		s.lsbIndex = p.BusInt.Read8(addr)
		s.msbIndex = p.BusInt.Read8(addr + 8)

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
	evalScan := p.scanLine
	_, yLen := p.getSpriteSize()
	for i := uint16(0); i < 64; i++ {

		// 0 yPos, 1 index, 2 attr, 3 xPos => i*4
		yPos := p.rOAM.Read8(i * 4)
		yPosEnd := uint16(yPos) + uint16(yLen)

		// if the scanLine intersects the sprite, it's a "hit"
		// copy sprite to the secondary OAM
		if evalScan >= int(yPos) && evalScan < int(yPosEnd) {
			p.sOAM[spriteCount].yPos = yPos
			p.sOAM[spriteCount].tIndex = p.rOAM.Read8(i*4 + 1)
			p.sOAM[spriteCount].attributes = p.rOAM.Read8(i*4 + 2)
			p.sOAM[spriteCount].xPos = p.rOAM.Read8(i*4 + 3)
			p.sOAM[spriteCount].id = uint8(i)

			spriteCount += 1
			if spriteCount >= 8 {
				p.regs[PPUSTATUS].Set(statusSpriteOverflow)
				break
			}
		}
	}
}

func (p *Ppu) clearOAM() {
	p.clearSecOAM()
	p.pOAM = p.sOAM
}

func (p *Ppu) clearSecOAM() {
	for i := range p.sOAM {
		// set back defaults
		p.sOAM[i] = OamSprite{
			yPos:       0xFF,
			tIndex:     0xFF,
			attributes: 0xFF,
			xPos:       0xFF,
			id:         64,
			lsbIndex:   0x00,
			msbIndex:   0x00,
		}
	}
}

func (p *Ppu) tick() {
	p.clock++
	p.exec()
}

func (p *Ppu) Ticks(nTicks int) {

	for i := 0; i < nTicks; i++ {
		p.tick()
	}
}

// BusInt
func (p *Ppu) Read8(addr uint16) uint8 {
	if addr < 0x4000 {
		// incomplete decoding means 0x2000-0x2007 are mirrored every 8 bytes
		addr = 0x2000 + addr%8
	}

	switch addr {
	// PPU Status (PPUSTATUS) - RDONLY
	case 0x2002:
		return p.regs[PPUSTATUS].Read()
	// PPU OAM Data (OAMDATA)
	case 0x2004:
		return p.regs[OAMDATA].Read()
	// PPU Data (PPUDATA)
	case 0x2007:
		return p.regs[PPUDATA].Read()
	}

	return 0
}

func (p *Ppu) Write8(addr uint16, val uint8) {

	p.setLastRegWrite(val)

	if addr < 0x4000 {
		// incomplete decoding means 0x2000-0x2007 are mirrored every 8 bytes
		addr = 0x2000 + addr%8
	}

	switch addr {
	// PPU Control (PPUCTRL) - WRONLY
	case 0x2000:
		p.regs[PPUCTRL].Write(val)
	// PPU Mask (PPUMASK) - WRONLY
	case 0x2001:
		p.regs[PPUMASK].Write(val)
	// PPU OAM Data (OAMADDR) - WRONLY
	case 0x2003:
		p.regs[OAMADDR].Write(val)
	// PPU OAM Data (OAMDATA)
	case 0x2004:
		p.regs[OAMDATA].Write(val)
	// PPU Scrolling (PPUSCROLL) - WRONLY
	case 0x2005:
		p.regs[PPUSCROLL].Write(val)
	// PPU Address (PPUADDR) - WRONLY
	case 0x2006:
		p.regs[PPUADDR].Write(val)
	// PPU Data (PPUDATA)
	case 0x2007:
		p.regs[PPUDATA].Write(val)
	// PPU OAM DMA (OAMDMA) - WRONLY
	case 0x4014:
		// handled by the dma engine
		panic("OAMDMA should have gone to the dma engine!")
	}
}
