package gones

const (
	PPUCTRL = iota
	PPUMASK
	PPUSTATUS
	OAMADDR
	OAMDATA
	PPUSCROLL
	PPUADDR
	PPUDATA
	OAMDMA
)

/* PPUCTRL
7  bit  0
---- ----
VPHB SINN
|||| ||||
|||| ||++- Base nametable address
|||| ||    (0 = $2000; 1 = $2400; 2 = $2800; 3 = $2C00)
|||| |+--- VRAM address increment per CPU read/write of PPUDATA
|||| |     (0: add 1, going across; 1: add 32, going down)
|||| +---- Sprite pattern table address for 8x8 sprites
||||       (0: $0000; 1: $1000; ignored in 8x16 mode)
|||+------ Background pattern table address (0: $0000; 1: $1000)
||+------- Sprite size (0: 8x8 pixels; 1: 8x16 pixels)
|+-------- PPU master/slave select
|          (0: read backdrop from EXT pins; 1: output color on EXT pins)
+--------- Generate an NMI at the start of the
           vertical blanking interval (0: off; 1: on)
*/
func (p *Ppu) getBaseNameTable() uint16 {
	return 0x2000 + uint16(p.regs[PPUCTRL].val&0x3)*0x400
}

func (p *Ppu) getVRAMAddrInc() uint16 {
	if p.regs[PPUCTRL].val&4 == 0 {
		return 1
	} else {
		return 32
	}
}

func (p *Ppu) getSpritePattern() uint16 {
	_, spriteYSize := p.getSpriteSize()
	if spriteYSize == 8 {
		return (uint16(p.regs[PPUCTRL].read()&8) >> 3) * 0x1000
	}
	return 0x1000
}

func (p *Ppu) getBackgroundTable() uint16 {
	return (uint16(p.regs[PPUCTRL].val&16) >> 4) * 0x1000
}

func (p *Ppu) getSpriteSize() (uint8, uint8) {
	return 8, (((p.regs[PPUCTRL].val >> 5) & 0x1) * 8) + 8
}

func (p *Ppu) getMasterSlaveSelect() uint8 {
	return (p.regs[PPUCTRL].val & 64) >> 6
}

func (p *Ppu) getNMIVertical() uint8 {
	return (p.regs[PPUCTRL].val & 128) >> 7
}

/* PPU Mask
7  bit  0
---- ----
BGRs bMmG
|||| ||||
|||| |||+- Greyscale (0: normal color, 1: produce a greyscale display)
|||| ||+-- 1: Show background in leftmost 8 pixels of screen, 0: Hide
|||| |+--- 1: Show sprites in leftmost 8 pixels of screen, 0: Hide
|||| +---- 1: Show background
|||+------ 1: Show sprites
||+------- Emphasize red
|+-------- Emphasize green
+--------- Emphasize blue
*/

func (p *Ppu) getGreyScale() uint8 {
	return p.regs[PPUMASK].val & 1
}

func (p *Ppu) showBackgroundLeft() uint8 {
	return (p.regs[PPUMASK].val & 2) >> 1
}

func (p *Ppu) showSpritesLeft() uint8 {
	return (p.regs[PPUMASK].val & 4) >> 2
}

func (p *Ppu) showBackground() bool {
	return ((p.regs[PPUMASK].val & 8) >> 3) == 1
}

func (p *Ppu) showSprites() bool {
	return ((p.regs[PPUMASK].val & 16) >> 4) == 1
}

// 0 R G B
func (p *Ppu) showEmphasize() uint8 {
	return (p.regs[PPUMASK].val & 0xE0) >> 5
}

/* PPUSTATUS
7  bit  0
---- ----
VSO. ....
|||| ||||
|||+-++++- Least significant bits previously written into a PPU register
|||        (due to register not being updated for this address)
||+------- Sprite overflow. The intent was for this flag to be set
||         whenever more than eight sprites appear on a scanline, but a
||         hardware bug causes the actual behavior to be more complicated
||         and generate false positives as well as false negatives; see
||         PPU sprite evaluation. This flag is set during sprite
||         evaluation and cleared at dot 1 (the second dot) of the
||         pre-render line.
|+-------- Sprite 0 Hit.  Set when a nonzero pixel of sprite 0 overlaps
|          a nonzero background pixel; cleared at dot 1 of the pre-render
|          line.  Used for raster timing.
+--------- Vertical blank has started (0: not in vblank; 1: in vblank).
           Set at dot 1 of line 241 (the line *after* the post-render
           line); cleared after reading $2002 and at dot 1 of the
           pre-render line.
*/

const (
	statusSpriteOverflow = 1 << 5
	statusSprite0Hit     = 1 << 6
)

func (p *Ppu) setSTATUSbits(val uint8) {
	p.regs[PPUSTATUS].set(val)
}

func (p *Ppu) setLastRegWrite(val uint8) {
	// Least significant bits previously written into a PPU register
	p.regs[PPUSTATUS].val = (p.regs[PPUSTATUS].val & 0xE0) | (val & 0x1F)
}

func (p *Ppu) readPPUStatus() uint8 {
	val := p.regs[PPUSTATUS].val

	p.clear(cpuIntNMI) // clear vblank
	p.wToggle.val = 0
	// Race Condition Warning: Reading PPUSTATUS within two cycles of the start of vertical blank will return 0
	// in bit 7 but clear the latch anyway, causing NMI to not occur that frame. See NMI and PPU_frame_timing for details.
	return val
}

func (p *Ppu) writePPUScroll() {
	val := p.regs[PPUSCROLL].read()
	if p.wToggle.val == 0 {
		// t: ....... ...HGFED = d: HGFED...
		// x:              CBA = d: .....CBA
		// w:                  = 1
		p.tRAM.val = (p.tRAM.val & 0xFFE0) | uint16(val>>8)
		p.xFine.write(val & 0x7)
		p.xScroll = val
		p.wToggle.val = 1
	} else {
		// t: CBA..HG FED..... = d: HGFEDCBA
		// w:                  = 01
		p.tRAM.val = (p.tRAM.val & 0x8FFF) | ((uint16(val) & 0x7) << 12)
		p.tRAM.val = (p.tRAM.val & 0xFC1F) | ((uint16(val) & 0xF8) << 2)
		p.wToggle.val = 0
	}
}

func (p *Ppu) writePPUAddr() {
	val := p.regs[PPUADDR].read()
	if p.wToggle.val == 0 {
		// http://wiki.nesdev.com/w/index.php/PPU_scrolling:
		// t: .FEDCBA ........ = d: ..FEDCBA
		// t: X...... ........ = 0
		// w:                  = 1
		p.tRAM.val = (p.tRAM.val & 0x80FF) | ((uint16(val) & 0x3F) << 8)
		p.wToggle.val = 1
	} else {
		// t: ....... HGFEDCBA = d: HGFEDCBA
		// v                   = t
		// w:                  = 0
		p.tRAM.val = (p.tRAM.val & 0xFF00) | uint16(val)
		p.vRAM.val = p.tRAM.val
		p.wToggle.val = 0
	}
}

func (p *Ppu) readPPUData() uint8 {
	val := p.busInt.read8(p.vRAM.val)
	p.regs[PPUDATA].val = val

	p.vRAM.val += p.getVRAMAddrInc()

	return val
}
func (p *Ppu) writePPUData() {
	val := p.regs[PPUDATA].val
	p.busInt.write8(p.vRAM.val, val)

	p.vRAM.val += p.getVRAMAddrInc()
}

func (p *Ppu) readOAMData() uint8 {
	addr := p.regs[OAMADDR].val
	val := p.rOAM.read8(uint16(addr))
	p.regs[OAMDATA].val = val

	p.regs[OAMADDR].val = addr + 1

	return val
}
func (p *Ppu) writeOAMData() {
	addr := p.regs[OAMADDR].val
	p.rOAM.write8(uint16(addr), p.regs[OAMDATA].val)
	p.regs[OAMADDR].val = addr + 1
}

func (p *Ppu) initRegisters() {

	p.regs[PPUCTRL].initx("PPUCTRL", 0, nil, nil)
	p.regs[PPUMASK].initx("PPUMASK", 0, nil, nil)
	p.regs[PPUSTATUS].initx("PPUSTATUS", 0, nil, p.readPPUStatus)
	p.regs[OAMADDR].initx("OAMADDR", 0, nil, nil)
	p.regs[OAMDATA].initx("OAMDATA", 0, p.writeOAMData, p.readOAMData)
	p.regs[PPUSCROLL].initx("PPUSCROLL", 0, p.writePPUScroll, nil)
	p.regs[PPUADDR].initx("PPUADDR", 0, p.writePPUAddr, nil)
	p.regs[PPUDATA].initx("PPUDATA", 0, p.writePPUData, p.readPPUData)
}
