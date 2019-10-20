package gones

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
	dataH uint8
	dataL uint8
}

type Ppu struct {
	busInt

	cycle    int
	scanLine int
	verbose  bool

	// cpu mapper registers
	regs [9]register

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

	interrupts iInterrupt
}

func (p *Ppu) init(busInt busInt, verbose bool, interrupts iInterrupt) {
	p.verbose = verbose
	p.busInt = busInt
	p.interrupts = interrupts
	p.cycle = 0
	p.scanLine = 0

	p.vRAM.init("v", 0)
	p.tRAM.init("t", 0)
	p.xFine.init("x", 0)
	p.wToggle.init("w", 0)
	p.rOAM.initf(256, 0xfe)

	p.initRegisters()
}

func (p *Ppu) reset() {
	p.init(p.busInt, p.verbose, p.interrupts)
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

func (p *Ppu) tick() {
	p.cycle += 1

	if p.cycle > 340 {
		p.scanLine += 1
		p.cycle = 0

		if p.scanLine == 241 {
			p.raise(cpuIntNMI)
		}

		if p.scanLine > 260 {
			p.scanLine = 0
			// may already be cleared as reading from PPSTATUS will do so
			p.clear(cpuIntNMI)
		}
	}

	if p.scanLine < 240 || p.scanLine == 261 {
		switch p.cycle {
		case 1:
			p.clearSecOAM() // if ( scanline == 261 ) { spriteOverflow = spriteHit = false; } break;
		case 257:
			p.evalSprites()
		case 321:
			p.loadSprites()
		}
	}
}

func (p *Ppu) loadSprites() {
	for i, _ := range p.sOAM {
		/*
		 // Copy secondary OAM into primary.
		        oam[i] = secOam[i];

		        // Different address modes depending on the sprite height:
		        if ( SpriteHeight() == 16 )
		            address = ((oam[i].tile & 1) * 0x1000) + ((oam[i].tile & ~1) * 16);
		        else
		            address = ((((control & ControlFlag_SpriteAddress) >> 3) & 1) * 0x1000) + (oam[i].tile * 16);

		        // Line inside the sprite.
		        uint16 sprY = (scanline - oam[i].y) % SpriteHeight();
		        // Vertical flip.
		        if ( oam[i].attribute & 0x80 )
		            sprY ^= SpriteHeight() - 1;

		        // Select the second tile if on 8x16.
		        address += sprY + (sprY & 8);

		        oam[i].dataL = memoryController->PpuRead( address + 0 );
		        oam[i].dataH = memoryController->PpuRead( address + 8 );
		*/
		p.pOAM[i] = p.sOAM[i]

		// very simple to test
		addr := 0x1000 + uint16(p.pOAM[i].tIndex*16)

		p.pOAM[i].dataL = p.busInt.read8(addr)
		p.pOAM[i].dataH = p.busInt.read8(addr + 8)
	}
}

func (p *Ppu) evalSprites() {
	spriteCount := 0
	for i := uint16(0); i < 64; i++ {

		yPos := p.rOAM.read8(i*4 + 0)
		_, yLen := p.getSpriteSize()
		yPosEnd := yPos + yLen

		// if the scanLine intersects the sprite, it's a "hit"
		// copy sprite to the secondary OAM
		if yPosEnd > yPos && p.scanLine >= int(yPos) && p.scanLine <= int(yPosEnd) {
			p.sOAM[spriteCount].yPos = p.rOAM.read8(i*4 + 0)
			p.sOAM[spriteCount].tIndex = p.rOAM.read8(i*4 + 1)
			p.sOAM[spriteCount].attributes = p.rOAM.read8(i*4 + 2)
			p.sOAM[spriteCount].xPos = p.rOAM.read8(i*4 + 3)

			spriteCount += 1
			if spriteCount >= 8 {
				p.setSTATUSbits(statusSpriteOverflow)
				break
			}
		}
	}
}

func (p *Ppu) clearSecOAM() {
	for i, _ := range p.sOAM {
		// set back to defaults
		p.sOAM[i] = OamSprite{
			yPos:       0xFF,
			tIndex:     0xFF,
			attributes: 0xFF,
			xPos:       0xFF,
		}
	}
}

func (p *Ppu) clock() {

	// 3 ppu ticks per 1 cpu
	p.exec()
	p.exec()
	p.exec()
}

func (p *Ppu) exec() {

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
		return p.getSTATUS()
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
		p.regs[OAMDMA].write(val)
	}
}
